package membership

// Player-initiated payments: the club's bank account (set by a Roaster admin),
// receipt uploads from players, and admin review. One receipt can cover monthly
// dues (one or more months), yellow cards and red cards together. Confirming a
// receipt records each item through the same pay_due / pay_cards commands admins
// use, so the register, audit history and allocation rules stay identical.
// Receipt files live in the ReceiptStore (S3); the database keeps the record.

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dilly3/edehson-league/adminauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentAccount struct {
	ID            int       `gorm:"primaryKey" json:"-"`
	BankName      string    `gorm:"size:60;not null" json:"bank_name"`
	AccountName   string    `gorm:"size:80;not null" json:"account_name"`
	AccountNumber string    `gorm:"size:10;not null" json:"account_number"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (PaymentAccount) TableName() string { return "club_payment_account" }

const (
	RequestPending   = "pending"
	RequestConfirmed = "confirmed"
	RequestRejected  = "rejected"

	maxReceiptBytes     = 5 << 20
	maxPendingPerMember = 3
)

type PaymentRequest struct {
	Base
	MemberID    string     `gorm:"type:uuid;not null;index" json:"member_id"`
	Member      Member     `gorm:"foreignKey:MemberID;constraint:OnDelete:RESTRICT" json:"-"`
	MemberName  string     `gorm:"size:80;not null" json:"member_name"` // name at upload time
	MonthsCSV   string     `gorm:"column:months;size:200;not null" json:"-"`
	Months      []string   `gorm:"-" json:"months"`
	YellowCount int        `gorm:"not null" json:"yellow_count"`
	RedCount    int        `gorm:"not null" json:"red_count"`
	DuesAmount  int64      `gorm:"not null" json:"dues_amount"`
	YellowAmt   int64      `gorm:"column:yellow_amount;not null" json:"yellow_amount"`
	RedAmt      int64      `gorm:"column:red_amount;not null" json:"red_amount"`
	Amount      int64      `gorm:"not null" json:"amount"`      // expected total at upload time
	AmountSent  int64      `gorm:"not null" json:"amount_sent"` // what the player says they transferred
	PaidOn      string     `gorm:"size:10;not null" json:"paid_on"`
	Note        string     `gorm:"size:300;not null" json:"note"`
	Status      string     `gorm:"size:10;not null;index;check:club_payreq_status,status IN ('pending','confirmed','rejected')" json:"status"`
	ReceiptKey  string     `gorm:"size:200;not null" json:"-"`
	ReceiptName string     `gorm:"size:120;not null" json:"receipt_name"`
	ReceiptType string     `gorm:"size:40;not null" json:"receipt_type"`
	ReceiptSize int        `gorm:"not null" json:"receipt_size"`
	ReviewNote  string     `gorm:"size:300;not null" json:"review_note"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	ReviewedBy  string     `gorm:"size:40;not null;default:''" json:"reviewed_by"`
	// The S3 file can be deleted by the super admin to save storage; the record stays.
	ReceiptDeletedAt *time.Time `json:"receipt_deleted_at"`
}

func (PaymentRequest) TableName() string { return "club_payment_requests" }

func (r *PaymentRequest) AfterFind(*gorm.DB) error {
	r.Months = splitMonths(r.MonthsCSV)
	return nil
}

func splitMonths(csv string) []string {
	if csv == "" {
		return []string{}
	}
	return strings.Split(csv, ",")
}

// ReceiptSettings holds options the super admin can change from the app.
type ReceiptSettings struct {
	ID            int       `gorm:"primaryKey" json:"-"`
	RetentionDays int       `gorm:"not null" json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ReceiptSettings) TableName() string { return "club_receipt_settings" }

func migratePaymentRequests(db *gorm.DB) error {
	// An early build stored one item type per receipt (kind, count) and kept the file
	// in the database (receipt). AutoMigrate never drops columns, and those were NOT
	// NULL, so remove them before migrating to the current shape.
	if err := db.Exec(`ALTER TABLE IF EXISTS club_payment_requests
		DROP CONSTRAINT IF EXISTS club_payreq_kind,
		DROP COLUMN IF EXISTS kind,
		DROP COLUMN IF EXISTS count,
		DROP COLUMN IF EXISTS receipt`).Error; err != nil {
		return err
	}
	return db.AutoMigrate(&PaymentAccount{}, &PaymentRequest{}, &ReceiptSettings{})
}

const maxRetentionDays = 3650

// retentionDays is the super admin's saved setting, or the startup default.
func retentionDays(ctx context.Context, db *gorm.DB) (int, error) {
	var s ReceiptSettings
	err := db.WithContext(ctx).Take(&s, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return int(ReceiptRetention / (24 * time.Hour)), nil
	}
	return s.RetentionDays, err
}

var accountNumberRE = regexp.MustCompile(`^[0-9]{10}$`)

var allowedReceiptTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "application/pdf": true}

// cardQuote prices `count` outstanding cards of one colour, oldest first,
// exactly as pay_cards allocates them. ok=false if fewer are outstanding.
func cardQuote(tx *gorm.DB, memberID, color string, count int) (total int64, ok bool, err error) {
	if count == 0 {
		return 0, true, nil
	}
	var cards []Card
	if err := tx.Where("member_id = ? AND color = ? AND void = ?", memberID, color, false).Order("occurred_on, created_at, id").Find(&cards).Error; err != nil {
		return 0, false, err
	}
	remaining := count
	for _, c := range cards {
		paid, err := paidCount(tx, c.ID)
		if err != nil {
			return 0, false, err
		}
		n := min(c.Count-paid, remaining)
		if n <= 0 {
			continue
		}
		total += int64(n) * c.UnitAmount
		remaining -= n
		if remaining == 0 {
			return total, true, nil
		}
	}
	return total, false, nil
}

type claims struct {
	months      map[string]bool
	yellow, red int
}

// pendingClaims returns months and card counts already awaiting review for a member.
func pendingClaims(tx *gorm.DB, memberID string) (claims, error) {
	var reqs []PaymentRequest
	out := claims{months: map[string]bool{}}
	if err := tx.Select("months", "yellow_count", "red_count").Where("member_id = ? AND status = ?", memberID, RequestPending).Find(&reqs).Error; err != nil {
		return out, err
	}
	for _, r := range reqs {
		for _, m := range splitMonths(r.MonthsCSV) {
			out.months[m] = true
		}
		out.yellow += r.YellowCount
		out.red += r.RedCount
	}
	return out, nil
}

type priced struct{ dues, yellow, red int64 }

func (p priced) total() int64 { return p.dues + p.yellow + p.red }

// priceRequest validates what the player is paying for and prices each part.
func priceRequest(tx *gorm.DB, memberID string, months []string, yellow, red int) (priced, error) {
	var out priced
	if len(months) == 0 && yellow == 0 && red == 0 {
		return out, invalid("Choose what this receipt pays for: monthly dues, yellow cards or red cards")
	}
	if len(months) > 24 || yellow < 0 || red < 0 || yellow > 1000 || red > 1000 {
		return out, invalid("Check the months and card counts")
	}
	pending, err := pendingClaims(tx, memberID)
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, m := range months {
		if !validMonth(m) || seen[m] {
			return out, invalid("Select valid months")
		}
		seen[m] = true
		if pending.months[m] {
			return out, conflict("A receipt for " + m + " is already awaiting review")
		}
		var d Due
		err := tx.Where("member_id = ? AND month = ?", memberID, m).Take(&d).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && d.Status != "unpaid") {
			return out, conflict(m + " has no unpaid dues. Refresh and choose again")
		}
		if err != nil {
			return out, err
		}
		out.dues += d.Amount
	}
	for _, card := range []struct {
		color          string
		count, pending int
		into           *int64
	}{{"yellow", yellow, pending.yellow, &out.yellow}, {"red", red, pending.red, &out.red}} {
		if card.count == 0 {
			continue
		}
		withNew, ok, err := cardQuote(tx, memberID, card.color, card.pending+card.count)
		if err != nil {
			return out, err
		}
		if !ok {
			return out, conflict("That is more " + card.color + " cards than are unpaid (including receipts awaiting review)")
		}
		already, _, err := cardQuote(tx, memberID, card.color, card.pending)
		if err != nil {
			return out, err
		}
		*card.into = withNew - already
	}
	return out, nil
}

func parseKobo(s string) (int64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" || len(frac) > 2 {
		return 0, false
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || w < 0 || w > 100000000 {
		return 0, false
	}
	var f int64
	if frac != "" {
		f, err = strconv.ParseInt(frac+strings.Repeat("0", 2-len(frac)), 10, 64)
		if err != nil {
			return 0, false
		}
	}
	return w*100 + f, w*100+f > 0
}

func registerPaymentRequests(g *gin.RouterGroup, db *gorm.DB, admin, superAdmin gin.HandlerFunc) {
	fail := func(c *gin.Context, err error, what string) {
		status, msg := explain(err)
		if status == 500 {
			log.Printf("%s: %v", what, err)
			msg = "Could not complete the request. Try again"
		}
		c.JSON(status, gin.H{"error": msg})
	}

	// ---- Club bank account ----
	g.GET("/payment-account", func(c *gin.Context) {
		var a PaymentAccount
		err := db.WithContext(c.Request.Context()).Take(&a, 1).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(200, gin.H{"data": nil})
			return
		}
		if err != nil {
			fail(c, err, "Read payment account")
			return
		}
		c.JSON(200, gin.H{"data": a})
	})
	g.PUT("/payment-account", admin, func(c *gin.Context) {
		var in struct {
			BankName      string `json:"bank_name"`
			AccountName   string `json:"account_name"`
			AccountNumber string `json:"account_number"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(400, gin.H{"error": "Provide the bank, account name and account number"})
			return
		}
		a := PaymentAccount{ID: 1, BankName: strings.TrimSpace(in.BankName), AccountName: strings.TrimSpace(in.AccountName), AccountNumber: strings.ReplaceAll(strings.TrimSpace(in.AccountNumber), " ", "")}
		if n := utf8.RuneCountInString(a.BankName); n < 2 || n > 60 {
			c.JSON(400, gin.H{"error": "Enter the bank name"})
			return
		}
		if n := utf8.RuneCountInString(a.AccountName); n < 2 || n > 80 {
			c.JSON(400, gin.H{"error": "Enter the account name"})
			return
		}
		if !accountNumberRE.MatchString(a.AccountNumber) {
			c.JSON(400, gin.H{"error": "Account numbers have exactly 10 digits"})
			return
		}
		if err := db.WithContext(c.Request.Context()).Save(&a).Error; err != nil {
			fail(c, err, "Save payment account")
			return
		}
		c.JSON(200, gin.H{"data": a})
	})

	// ---- Player: upload a receipt (public; an admin must confirm) ----
	g.POST("/payment-requests", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxReceiptBytes+64<<10)
		if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
			c.JSON(400, gin.H{"error": "Receipts must be 5 MB or smaller"})
			return
		}
		memberID := c.PostForm("member_id")
		paidOn := c.PostForm("paid_on")
		note := strings.TrimSpace(c.PostForm("note"))
		yellow, _ := strconv.Atoi(c.DefaultPostForm("yellow_count", "0"))
		red, _ := strconv.Atoi(c.DefaultPostForm("red_count", "0"))
		var months []string
		if raw := strings.TrimSpace(c.PostForm("months")); raw != "" {
			months = strings.Split(raw, ",")
		}
		sort.Strings(months)
		if _, err := uuid.Parse(memberID); err != nil {
			c.JSON(400, gin.H{"error": "Select your name"})
			return
		}
		sent, ok := parseKobo(c.PostForm("amount_sent"))
		if !ok {
			c.JSON(400, gin.H{"error": "Enter the amount you transferred"})
			return
		}
		if !validDate(paidOn) || paidOn > today() || paidOn < "2026-01-01" {
			c.JSON(400, gin.H{"error": "Enter the date you made the transfer (not in the future)"})
			return
		}
		if utf8.RuneCountInString(note) > 300 {
			c.JSON(400, gin.H{"error": "Notes must contain at most 300 characters"})
			return
		}
		file, header, err := c.Request.FormFile("receipt")
		if err != nil {
			c.JSON(400, gin.H{"error": "Attach your payment receipt"})
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxReceiptBytes+1))
		if err != nil || len(data) == 0 {
			c.JSON(400, gin.H{"error": "The receipt could not be read"})
			return
		}
		if len(data) > maxReceiptBytes {
			c.JSON(400, gin.H{"error": "Receipts must be 5 MB or smaller"})
			return
		}
		contentType := http.DetectContentType(data)
		if !allowedReceiptTypes[contentType] {
			c.JSON(400, gin.H{"error": "Upload the receipt as a JPG, PNG, WebP image or a PDF"})
			return
		}
		name := strings.TrimSpace(header.Filename)
		if utf8.RuneCountInString(name) > 120 {
			name = string([]rune(name)[:120])
		}

		ctx := c.Request.Context()
		// Validate first so we don't upload files for requests that will be refused.
		var member Member
		if err := db.WithContext(ctx).First(&member, "id = ?", memberID).Error; err != nil {
			fail(c, err, "Load member for payment request")
			return
		}
		if _, err := priceRequest(db.WithContext(ctx), member.ID, months, yellow, red); err != nil {
			fail(c, err, "Price payment request")
			return
		}
		id := uuid.NewString()
		key := receiptKey(member.Name, id, today()[:7], contentType)
		if err := receipts.Put(ctx, key, contentType, data); err != nil {
			log.Printf("Store receipt %s: %v", key, err)
			c.JSON(503, gin.H{"error": "The receipt could not be uploaded. Try again shortly"})
			return
		}
		var req PaymentRequest
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(7231941)").Error; err != nil {
				return err
			}
			var pending int64
			if err := tx.Model(&PaymentRequest{}).Where("member_id = ? AND status = ?", member.ID, RequestPending).Count(&pending).Error; err != nil {
				return err
			}
			if pending >= maxPendingPerMember {
				return conflict("You already have receipts awaiting review. Please wait for an admin to check them")
			}
			p, err := priceRequest(tx, member.ID, months, yellow, red) // re-check under the lock
			if err != nil {
				return err
			}
			req = PaymentRequest{MemberID: member.ID, MemberName: member.Name, MonthsCSV: strings.Join(months, ","), YellowCount: yellow, RedCount: red,
				DuesAmount: p.dues, YellowAmt: p.yellow, RedAmt: p.red, Amount: p.total(), AmountSent: sent, PaidOn: paidOn, Note: note,
				Status: RequestPending, ReceiptKey: key, ReceiptName: name, ReceiptType: contentType, ReceiptSize: len(data)}
			req.ID = id
			return tx.Create(&req).Error
		})
		if err != nil {
			if delErr := receipts.Delete(context.Background(), key); delErr != nil {
				log.Printf("Remove orphaned receipt %s: %v", key, delErr)
			}
			fail(c, err, "Save payment request")
			return
		}
		req.Months = splitMonths(req.MonthsCSV)
		c.JSON(201, gin.H{"data": req})
	})

	// Player-facing: what is awaiting review for one member (no receipts or notes).
	g.GET("/payment-requests/pending", func(c *gin.Context) {
		memberID := c.Query("member_id")
		if _, err := uuid.Parse(memberID); err != nil {
			c.JSON(400, gin.H{"error": "Select a member"})
			return
		}
		var reqs []PaymentRequest
		if err := db.WithContext(c.Request.Context()).Where("member_id = ? AND status = ?", memberID, RequestPending).Order("created_at").Find(&reqs).Error; err != nil {
			fail(c, err, "List pending payment requests")
			return
		}
		out := make([]gin.H, 0, len(reqs))
		for _, r := range reqs {
			out = append(out, gin.H{"id": r.ID, "months": r.Months, "yellow_count": r.YellowCount, "red_count": r.RedCount, "amount": r.Amount, "created_at": r.CreatedAt})
		}
		c.JSON(200, gin.H{"data": out})
	})

	// ---- Admin review ----
	g.GET("/payment-requests", admin, func(c *gin.Context) {
		status := c.DefaultQuery("status", RequestPending)
		q := db.WithContext(c.Request.Context()).Order("created_at DESC").Limit(1000)
		if status != "all" {
			q = q.Where("status = ?", status)
		}
		if member := c.Query("member_id"); member != "" {
			if _, err := uuid.Parse(member); err != nil {
				c.JSON(400, gin.H{"error": "Select a member"})
				return
			}
			q = q.Where("member_id = ?", member)
		}
		var reqs []PaymentRequest
		if err := q.Find(&reqs).Error; err != nil {
			fail(c, err, "List payment requests")
			return
		}
		c.JSON(200, gin.H{"data": reqs})
	})

	g.GET("/payment-requests/:id/receipt", admin, func(c *gin.Context) {
		var r PaymentRequest
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			c.JSON(404, gin.H{"error": "Receipt not found"})
			return
		}
		if err := db.WithContext(c.Request.Context()).First(&r, "id = ?", c.Param("id")).Error; err != nil {
			fail(c, err, "Read receipt record")
			return
		}
		if r.ReceiptDeletedAt != nil {
			c.JSON(410, gin.H{"error": "This receipt file was deleted to save storage"})
			return
		}
		data, err := receipts.Get(c.Request.Context(), r.ReceiptKey)
		if err != nil {
			log.Printf("Read receipt %s: %v", r.ReceiptKey, err)
			c.JSON(502, gin.H{"error": "The receipt file could not be loaded"})
			return
		}
		c.Header("Content-Disposition", `inline; filename="receipt"`)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, r.ReceiptType, data)
	})

	// ---- Super admin: delete stored receipt files (records and payments are kept) ----
	g.DELETE("/payment-requests/:id/receipt", superAdmin, func(c *gin.Context) {
		id := c.Param("id")
		if _, err := uuid.Parse(id); err != nil {
			c.JSON(404, gin.H{"error": "Receipt not found"})
			return
		}
		var r PaymentRequest
		if err := db.WithContext(c.Request.Context()).First(&r, "id = ?", id).Error; err != nil {
			fail(c, err, "Load receipt for deletion")
			return
		}
		if r.Status == RequestPending {
			c.JSON(409, gin.H{"error": "Confirm or reject this receipt before deleting its file"})
			return
		}
		if r.ReceiptDeletedAt != nil {
			c.Status(204)
			return
		}
		if err := deleteReceiptFile(c.Request.Context(), db, r); err != nil {
			fail(c, err, "Delete receipt file")
			return
		}
		c.Status(204)
	})
	// How long reviewed receipt files are kept: any Roaster admin can read it,
	// only the super admin can change it.
	g.GET("/receipt-settings", admin, func(c *gin.Context) {
		days, err := retentionDays(c.Request.Context(), db)
		if err != nil {
			fail(c, err, "Read receipt settings")
			return
		}
		c.JSON(200, gin.H{"data": gin.H{"retention_days": days}})
	})
	g.PUT("/receipt-settings", superAdmin, func(c *gin.Context) {
		var in struct {
			RetentionDays int `json:"retention_days"`
		}
		if err := c.ShouldBindJSON(&in); err != nil || in.RetentionDays < 1 || in.RetentionDays > maxRetentionDays {
			c.JSON(400, gin.H{"error": "Keep receipts for between 1 and 3650 days"})
			return
		}
		s := ReceiptSettings{ID: 1, RetentionDays: in.RetentionDays}
		if err := db.WithContext(c.Request.Context()).Save(&s).Error; err != nil {
			fail(c, err, "Save receipt settings")
			return
		}
		c.JSON(200, gin.H{"data": gin.H{"retention_days": s.RetentionDays}})
	})

	g.POST("/payment-requests/purge-files", superAdmin, func(c *gin.Context) {
		var in struct {
			OlderThanDays int `json:"older_than_days"`
		}
		if err := c.ShouldBindJSON(&in); err != nil || in.OlderThanDays < 0 || in.OlderThanDays > 3650 {
			c.JSON(400, gin.H{"error": "Choose how many days old a reviewed receipt must be"})
			return
		}
		deleted, failed, freed, err := purgeReceiptFiles(c.Request.Context(), db, time.Now().UTC().AddDate(0, 0, -in.OlderThanDays))
		if err != nil {
			fail(c, err, "Purge receipt files")
			return
		}
		c.JSON(200, gin.H{"data": gin.H{"deleted": deleted, "failed": failed, "bytes_freed": freed}})
	})

	g.POST("/payment-requests/:id/confirm", admin, func(c *gin.Context) {
		id := c.Param("id")
		if _, err := uuid.Parse(id); err != nil {
			c.JSON(404, gin.H{"error": "Receipt not found"})
			return
		}
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			return confirmRequest(tx, id, reviewer(c))
		})
		if err != nil {
			fail(c, err, "Confirm payment request")
			return
		}
		c.Status(204)
	})

	g.POST("/payment-requests/:id/reject", admin, func(c *gin.Context) {
		var in struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&in)
		reason := strings.TrimSpace(in.Reason)
		if reason == "" || utf8.RuneCountInString(reason) > 300 {
			c.JSON(400, gin.H{"error": "Give a reason for rejecting the receipt"})
			return
		}
		now := time.Now().UTC()
		res := db.WithContext(c.Request.Context()).Model(&PaymentRequest{}).Where("id = ? AND status = ?", c.Param("id"), RequestPending).
			Updates(map[string]interface{}{"status": RequestRejected, "review_note": reason, "reviewed_at": now, "reviewed_by": reviewer(c)})
		if res.Error != nil {
			fail(c, res.Error, "Reject payment request")
			return
		}
		if res.RowsAffected == 0 {
			c.JSON(409, gin.H{"error": "This receipt has already been reviewed"})
			return
		}
		c.Status(204)
	})
}

// confirmRequest records every payment a pending receipt covers, then marks it confirmed.
// It runs inside the caller's transaction; each command becomes a savepoint, so any
// failure rolls the whole confirmation back.
func confirmRequest(tx *gorm.DB, id, reviewedBy string) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(7231941)").Error; err != nil {
		return err
	}
	var r PaymentRequest
	if err := tx.First(&r, "id = ?", id).Error; err != nil {
		return err
	}
	if r.Status != RequestPending {
		return conflict("This receipt has already been reviewed")
	}
	note := "Confirmed from player receipt"
	if r.Note != "" {
		note += ": " + r.Note
		if utf8.RuneCountInString(note) > 300 {
			note = string([]rune(note)[:300])
		}
	}
	for _, m := range splitMonths(r.MonthsCSV) {
		var d Due
		err := tx.Where("member_id = ? AND month = ?", r.MemberID, m).Take(&d).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && d.Status != "unpaid") {
			return conflict(m + " is no longer unpaid. Reject this receipt or correct the record first")
		}
		if err != nil {
			return err
		}
		if err := execute(tx, Input{RequestID: uuid.NewString(), Type: "pay_due", MemberID: r.MemberID, RecordID: d.ID, Revision: d.Revision, PaidOn: r.PaidOn, Note: note}); err != nil {
			return err
		}
		if err := tx.Model(&Payment{}).Where("due_id = ? AND void = false AND receipt_request_id IS NULL", d.ID).Update("receipt_request_id", r.ID).Error; err != nil {
			return err
		}
	}
	for _, card := range []struct {
		color string
		count int
	}{{"yellow", r.YellowCount}, {"red", r.RedCount}} {
		if card.count == 0 {
			continue
		}
		total, ok, err := cardQuote(tx, r.MemberID, card.color, card.count)
		if err != nil {
			return err
		}
		if !ok {
			return conflict("Fewer " + card.color + " cards are unpaid than this receipt covers. Reject it or correct the record first")
		}
		if err := execute(tx, Input{RequestID: uuid.NewString(), Type: "pay_cards", MemberID: r.MemberID, Color: card.color, Count: card.count, PaidOn: r.PaidOn, ExpectedAmount: total, Note: note}); err != nil {
			return err
		}
		// The card payment just created is the newest one for this member and colour
		// (writes are serialized by the register lock held for this transaction).
		var p Payment
		if err := tx.Where("member_id = ? AND kind = ? AND void = false AND receipt_request_id IS NULL", r.MemberID, card.color).Order("created_at DESC, id DESC").First(&p).Error; err != nil {
			return err
		}
		if err := tx.Model(&p).Update("receipt_request_id", r.ID).Error; err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	return tx.Model(&PaymentRequest{}).Where("id = ?", r.ID).Updates(map[string]interface{}{"status": RequestConfirmed, "reviewed_at": now, "reviewed_by": reviewedBy}).Error
}

// reviewer is the name shown in the payment history for whoever confirmed or rejected a receipt.
func reviewer(c *gin.Context) string {
	name := adminauth.PrincipalFrom(c).Username
	if name == "" {
		name = "Admin"
	}
	if utf8.RuneCountInString(name) > 40 {
		name = string([]rune(name)[:40])
	}
	return name
}

// deleteReceiptFile removes the stored file and marks the record. A file that is
// already missing from storage still counts as deleted.
func deleteReceiptFile(ctx context.Context, db *gorm.DB, r PaymentRequest) error {
	if err := receipts.Delete(ctx, r.ReceiptKey); err != nil && !isNotFound(err) {
		return err
	}
	now := time.Now().UTC()
	return db.WithContext(ctx).Model(&PaymentRequest{}).Where("id = ?", r.ID).Update("receipt_deleted_at", now).Error
}

func isNotFound(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	var nf interface{ ErrorCode() string }
	return errors.As(err, &nf) && (nf.ErrorCode() == "NoSuchKey" || nf.ErrorCode() == "NotFound")
}

// purgeReceiptFiles deletes stored files for receipts reviewed before cutoff.
func purgeReceiptFiles(ctx context.Context, db *gorm.DB, cutoff time.Time) (deleted, failed int, freed int64, err error) {
	var reqs []PaymentRequest
	if err := db.WithContext(ctx).Where("status <> ? AND receipt_deleted_at IS NULL AND reviewed_at <= ?", RequestPending, cutoff).Find(&reqs).Error; err != nil {
		return 0, 0, 0, err
	}
	for _, r := range reqs {
		if err := deleteReceiptFile(ctx, db, r); err != nil {
			log.Printf("Purge receipt %s: %v", r.ReceiptKey, err)
			failed++
			continue
		}
		deleted++
		freed += int64(r.ReceiptSize)
	}
	return deleted, failed, freed, nil
}

// ReceiptRetention is the default time reviewed (confirmed or rejected) receipt files
// are kept before automatic deletion, used until the super admin saves a setting.
// Records and payments are never deleted.
var ReceiptRetention = 90 * 24 * time.Hour

// ReceiptCleanupInterval is how often expired receipt files are removed.
const ReceiptCleanupInterval = 7 * 24 * time.Hour

// StartReceiptCleanup deletes expired receipt files now and then once a week.
func StartReceiptCleanup(ctx context.Context, db *gorm.DB) {
	run := func() {
		days, err := retentionDays(ctx, db)
		if err != nil {
			log.Printf("Receipt cleanup: read retention: %v", err)
			return
		}
		deleted, failed, freed, err := purgeReceiptFiles(ctx, db, time.Now().UTC().AddDate(0, 0, -days))
		if err != nil {
			log.Printf("Receipt cleanup: %v", err)
			return
		}
		if deleted > 0 || failed > 0 {
			log.Printf("Receipt cleanup: deleted %d files (%d KB), %d failed", deleted, freed/1024, failed)
		}
	}
	go func() {
		run()
		ticker := time.NewTicker(ReceiptCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
