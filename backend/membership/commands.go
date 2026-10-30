package membership

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
	"time"
	"unicode/utf8"
)

type Input struct {
	Year               int     `json:"year"`
	CardIndex          int     `json:"card_index"`
	ExpectedCount      int     `json:"expected_count"`
	ExpectedOccurredOn *string `json:"expected_occurred_on"`
	ExpectedPaidOn     *string `json:"expected_paid_on"`
	ExpectedNote       *string `json:"expected_note"`
	ExpectedAmount     int64   `json:"expected_amount"`
	RequestID          string  `json:"request_id"`
	Type               string  `json:"type"`
	MemberID           string  `json:"member_id"`
	RecordID           string  `json:"record_id"`
	Name               string  `json:"name"`
	JoinedOn           string  `json:"joined_on"`
	Active             bool    `json:"active"`
	Revision           int     `json:"revision"`
	Kind               string  `json:"kind"`
	EffectiveFrom      string  `json:"effective_from"`
	Amount             int64   `json:"amount"`
	Month              string  `json:"month"`
	Color              string  `json:"color"`
	Count              int     `json:"count"`
	OccurredOn         string  `json:"occurred_on"`
	PaidOn             string  `json:"paid_on"`
	Note               string  `json:"note"`
}

type problem struct {
	status  int
	message string
}

func (e *problem) Error() string { return e.message }
func invalid(s string) error     { return &problem{400, s} }
func conflict(s string) error    { return &problem{409, s} }
func today() string              { return time.Now().In(time.FixedZone("WAT", 3600)).Format("2006-01-02") }
func validDate(s string) bool {
	d, e := time.Parse("2006-01-02", s)
	return e == nil && d.Format("2006-01-02") == s
}
func validMonth(s string) bool {
	d, e := time.Parse("2006-01", s)
	return e == nil && d.Format("2006-01") == s && s >= "1900-01"
}
func snapshot(x interface{}) string { b, _ := json.Marshal(x); return string(b) }
func validName(s string) bool       { n := utf8.RuneCountInString(s); return n >= 2 && n <= 80 }
func rate(tx *gorm.DB, kind, date string) (int64, error) {
	var f Fee
	err := tx.Where("kind = ? AND effective_from <= ?", kind, date).Order("effective_from DESC").First(&f).Error
	return f.Amount, err
}

func execute(db *gorm.DB, in Input) error {
	if _, err := uuid.Parse(in.RequestID); err != nil {
		return invalid("A valid request_id is required")
	}
	if utf8.RuneCountInString(in.Note) > 300 {
		return invalid("Notes must contain at most 300 characters")
	}
	raw, _ := json.Marshal(in)
	hash := sha256.Sum256(raw)
	digest := hex.EncodeToString(hash[:])
	return db.Transaction(func(tx *gorm.DB) error {
		// The register is small: serializing its writes makes payments, voids and
		// issuance deterministic, including retries from two admin tabs.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(7231941)").Error; err != nil {
			return err
		}
		var previous CommandRecord
		err := tx.Where("request_id = ?", in.RequestID).First(&previous).Error
		if err == nil {
			if previous.Hash != digest {
				return conflict("Request ID already used for a different operation")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := ensureDuesTx(tx, today()); err != nil {
			return err
		}
		log := CommandRecord{RequestID: in.RequestID, Hash: digest, Type: in.Type, MemberID: in.MemberID, Note: strings.TrimSpace(in.Note)}
		if err := apply(tx, in, &log); err != nil {
			return err
		}
		if in.Type == "add_member" || in.Type == "edit_member" || in.Type == "set_annual_fee" || in.Type == "pay_due" || in.Type == "void_payment" {
			if err := ensureDuesTx(tx, today()); err != nil {
				return err
			}
		}
		return tx.Create(&log).Error
	})
}

func apply(tx *gorm.DB, in Input, log *CommandRecord) error {
	switch in.Type {
	case "add_member":
		name := strings.TrimSpace(in.Name)
		if !validName(name) || !validDate(in.JoinedOn) || in.JoinedOn < "1900-01-01" || in.JoinedOn > today() {
			return invalid("Provide a 2–80 character name and a valid joining date, not in the future")
		}
		m := Member{Name: name, JoinedOn: in.JoinedOn, Active: true}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		log.MemberID = m.ID
		log.After = snapshot(m)
		return nil
	case "set_annual_fee":
		if in.Year < 2026 || in.Year > 2200 || fmt.Sprintf("%04d", in.Year) < today()[:4] {
			return invalid("Choose the current year or a future year")
		}
		if in.Amount < 1 || in.Amount > 10000000000 {
			return invalid("Provide a positive monthly fee in kobo")
		}
		var count int64
		if err := tx.Model(&Due{}).Where("month >= ? AND month <= ?", fmt.Sprintf("%04d-01", in.Year), fmt.Sprintf("%04d-12", in.Year)).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return conflict("This year's fee is locked because dues have already been issued")
		}
		var fee AnnualFee
		err := tx.Where("year = ?", in.Year).First(&fee).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && fee.Revision != in.Revision {
			return conflict("Annual fee changed. Refresh before saving")
		}
		log.Before = snapshot(fee)
		fee.Year = in.Year
		fee.Amount = in.Amount
		fee.Revision++
		if err := tx.Save(&fee).Error; err != nil {
			return err
		}
		log.After = snapshot(fee)
		return nil
	case "set_fee":
		if in.Kind == "monthly" {
			return invalid("Set the annual monthly dues rate instead")
		}
		if in.Kind != "monthly" && in.Kind != "yellow" && in.Kind != "red" {
			return invalid("Unknown fee kind")
		}
		if !validDate(in.EffectiveFrom) || in.EffectiveFrom < today() {
			return invalid("Fee changes must take effect today or later")
		}
		if in.Kind == "monthly" && (in.EffectiveFrom < nextMonth(today()[:7]) || in.EffectiveFrom[8:] != "01") {
			return invalid("Monthly fee changes must start on the first day of a future month")
		}
		if in.Amount < 1 || in.Amount > 10000000000 {
			return invalid("Fee must be positive and no more than ₦100,000,000, in kobo")
		}
		var f Fee
		err := tx.Where("kind = ? AND effective_from = ?", in.Kind, in.EffectiveFrom).First(&f).Error
		if err == nil {
			return conflict("A fee already exists for that date; choose another effective date")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		f = Fee{Kind: in.Kind, EffectiveFrom: in.EffectiveFrom, Amount: in.Amount}
		if err := tx.Create(&f).Error; err != nil {
			return err
		}
		log.After = snapshot(f)
		return nil
	case "issue_dues":
		return invalid("Monthly dues are now issued automatically")
	}
	if _, err := uuid.Parse(in.MemberID); err != nil {
		return invalid("Select a member")
	}
	var member Member
	if err := tx.First(&member, "id = ?", in.MemberID).Error; err != nil {
		return err
	}
	switch in.Type {
	case "edit_member":
		name := strings.TrimSpace(in.Name)
		if !validName(name) {
			return invalid("Member name must contain 2–80 characters")
		}
		if member.Revision != in.Revision {
			return conflict("This member changed. Refresh before saving")
		}
		joinedOn := member.JoinedOn
		// Older clients omit joined_on when editing names/status.
		if in.JoinedOn != "" {
			if !validDate(in.JoinedOn) || in.JoinedOn < "1900-01-01" || in.JoinedOn > today() {
				return invalid("Provide a valid joining date, not in the future")
			}
			joinedOn = in.JoinedOn
		}
		log.Before = snapshot(member)
		member.Name = name
		member.JoinedOn = joinedOn
		member.Active = in.Active
		member.Revision++
		if err := tx.Save(&member).Error; err != nil {
			return err
		}
		log.After = snapshot(member)
		return nil
	case "add_cards":
		if !member.Active {
			return conflict("Reactivate this member before adding cards")
		}
		if (in.Color != "yellow" && in.Color != "red") || in.Count < 1 || in.Count > 10000 || !validDate(in.OccurredOn) || in.OccurredOn < member.JoinedOn || in.OccurredOn > today() {
			return invalid("Provide a card color, 1–10,000 cards, and a date on or after joining and not in the future")
		}
		amount, err := rate(tx, in.Color, in.OccurredOn)
		if err != nil {
			return err
		}
		c := Card{MemberID: member.ID, Color: in.Color, Count: in.Count, UnitAmount: amount, OccurredOn: in.OccurredOn, Note: strings.TrimSpace(in.Note)}
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		log.After = snapshot(c)
		return nil
	case "pay_cards":
		if (in.Color != "yellow" && in.Color != "red") || in.Count < 1 || in.Count > 10000 || !validDate(in.PaidOn) || in.PaidOn > today() {
			return invalid("Provide a card color, whole-card count and valid payment date")
		}
		var cards []Card
		if err := tx.Where("member_id = ? AND color = ? AND void = ?", member.ID, in.Color, false).Order("occurred_on, created_at, id").Find(&cards).Error; err != nil {
			return err
		}
		var allocations []Allocation
		remaining := in.Count
		var total int64
		for _, c := range cards {
			paid, err := paidCount(tx, c.ID)
			if err != nil {
				return err
			}
			available := c.Count - paid
			if available <= 0 {
				continue
			}
			if c.OccurredOn > in.PaidOn {
				return invalid("Payment date cannot precede the cards being settled")
			}
			n := min(available, remaining)
			allocations = append(allocations, Allocation{CardID: c.ID, Count: n})
			total += int64(n) * c.UnitAmount
			remaining -= n
			if remaining == 0 {
				break
			}
		}
		if remaining != 0 {
			return conflict("Payment exceeds the number of outstanding cards; refresh the register")
		}
		if in.ExpectedAmount != total {
			return conflict("Card settlement amount changed. Refresh before recording payment")
		}
		p := Payment{MemberID: member.ID, Kind: in.Color, Amount: total, PaidOn: in.PaidOn, Note: strings.TrimSpace(in.Note)}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		for i := range allocations {
			allocations[i].PaymentID = p.ID
			if err := tx.Create(&allocations[i]).Error; err != nil {
				return err
			}
		}
		log.After = snapshot(p)
		return nil
	case "pay_due", "waive_due", "reopen_due":
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select an issued monthly due")
		}
		var d Due
		if err := tx.First(&d, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if d.Revision != in.Revision {
			return conflict("This due changed. Refresh before saving")
		}
		log.Before = snapshot(d)
		if in.Type == "reopen_due" && d.AutoWaived {
			return conflict("This month is waived automatically because it precedes the first paid dues month. Correct the payment records to recalculate it")
		}
		if in.Type == "reopen_due" {
			if d.Status != "waived" || strings.TrimSpace(in.Note) == "" {
				return invalid("Only waived dues can be reopened; provide a reason")
			}
			d.Status = "unpaid"
		} else {
			if d.Status != "unpaid" {
				return conflict("This due has already been paid or waived")
			}
			if in.Type == "waive_due" {
				if strings.TrimSpace(in.Note) == "" {
					return invalid("Provide a waiver reason")
				}
				d.Status = "waived"
			} else {
				if !validDate(in.PaidOn) || in.PaidOn > today() || in.PaidOn < earliestDuesPayment(d.Month, member.JoinedOn) {
					return invalid("Provide a payment date from the joining month, not in the future")
				}
				id := d.ID
				p := Payment{MemberID: member.ID, Kind: "monthly", DueID: &id, Amount: d.Amount, PaidOn: in.PaidOn, Note: strings.TrimSpace(in.Note)}
				if err := tx.Create(&p).Error; err != nil {
					return err
				}
				d.Status = "paid"
			}
		}
		d.Revision++
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		log.After = snapshot(d)
		return nil
	case "void_card":
		if strings.TrimSpace(in.Note) == "" {
			return invalid("Provide a correction reason")
		}
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select a card record")
		}
		var c Card
		if err := tx.First(&c, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if c.Void {
			return conflict("Card record is already void")
		}
		n, err := paidCount(tx, c.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			return conflict("Void payments against these cards before correcting the card record")
		}
		log.Before = snapshot(c)
		c.Void = true
		if err := tx.Save(&c).Error; err != nil {
			return err
		}
		log.After = snapshot(c)
		return nil
	case "edit_card":
		return editIndividualCard(tx, in, member, log)
	case "edit_dues_payment":
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select a monthly dues payment")
		}
		var p Payment
		if err := tx.First(&p, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if p.Void || p.Kind != "monthly" || p.DueID == nil {
			return conflict("Only active monthly dues payments can be edited")
		}
		if in.ExpectedPaidOn == nil || in.ExpectedNote == nil || p.PaidOn != *in.ExpectedPaidOn || p.Note != *in.ExpectedNote {
			return conflict("Payment changed. Refresh before saving")
		}
		var d Due
		if err := tx.First(&d, "id = ? AND member_id = ?", *p.DueID, member.ID).Error; err != nil {
			return err
		}
		if d.Status != "paid" {
			return conflict("This due is no longer paid")
		}
		if !validDate(in.PaidOn) || in.PaidOn > today() || in.PaidOn < earliestDuesPayment(d.Month, member.JoinedOn) {
			return invalid("Choose a payment date from the joining month, not in the future")
		}
		log.Before = snapshot(p)
		p.PaidOn = in.PaidOn
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		log.After = snapshot(p)
		return nil
	case "edit_card_payment":
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select a card payment")
		}
		if !validDate(in.PaidOn) || in.PaidOn < member.JoinedOn || in.PaidOn > today() {
			return invalid("Choose a payment date on or after joining and not in the future")
		}
		var p Payment
		if err := tx.First(&p, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if (p.Kind != "yellow" && p.Kind != "red") || p.Void {
			return conflict("Only active card payments can be edited")
		}
		if in.ExpectedPaidOn == nil || in.ExpectedNote == nil || p.PaidOn != *in.ExpectedPaidOn || p.Note != *in.ExpectedNote {
			return conflict("This payment changed. Refresh before saving")
		}
		var cards []Card
		if err := tx.Joins("JOIN club_allocations a ON a.card_id = club_cards.id").Where("a.payment_id = ?", p.ID).Find(&cards).Error; err != nil {
			return err
		}
		if len(cards) == 0 {
			return conflict("This payment has no card allocations; review the record before editing")
		}
		for _, c := range cards {
			if c.OccurredOn != "" && in.PaidOn < c.OccurredOn {
				return invalid("Payment date cannot be before a linked card's issue date")
			}
		}
		log.Before = snapshot(p)
		p.PaidOn = in.PaidOn
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		log.After = snapshot(p)
		return nil
	case "edit_donation":
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select a donation")
		}
		if !validDate(in.PaidOn) || in.PaidOn < member.JoinedOn || in.PaidOn > today() {
			return invalid("Choose a payment date on or after joining and not in the future")
		}
		purpose := strings.TrimSpace(in.Note)
		if purpose == "" {
			return invalid("Enter the donation purpose")
		}
		var p Payment
		if err := tx.First(&p, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if p.Kind != "donation" || p.Void {
			return conflict("Only active donations can be edited")
		}
		if in.ExpectedPaidOn == nil || in.ExpectedNote == nil || p.PaidOn != *in.ExpectedPaidOn || p.Note != *in.ExpectedNote {
			return conflict("This donation changed. Refresh before saving")
		}
		log.Before = snapshot(p)
		p.PaidOn = in.PaidOn
		p.Note = purpose
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		log.After = snapshot(p)
		return nil
	case "void_payment":
		if strings.TrimSpace(in.Note) == "" {
			return invalid("Provide a correction reason")
		}
		if _, err := uuid.Parse(in.RecordID); err != nil {
			return invalid("Select a payment")
		}
		var p Payment
		if err := tx.First(&p, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
			return err
		}
		if p.Void {
			return conflict("Payment is already void")
		}
		log.Before = snapshot(p)
		p.Void = true
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if p.DueID != nil {
			if err := tx.Model(&Due{}).Where("id = ?", *p.DueID).Updates(map[string]interface{}{"status": "unpaid", "revision": gorm.Expr("revision + 1")}).Error; err != nil {
				return err
			}
		}
		log.After = snapshot(p)
		return nil
	}
	return invalid("Unknown membership operation")
}
func nextMonth(month string) string {
	d, _ := time.Parse("2006-01", month)
	return d.AddDate(0, 1, 0).Format("2006-01-02")
}
func paidCount(tx *gorm.DB, cardID string) (int, error) {
	var n int
	err := tx.Table("club_allocations AS a").Select("COALESCE(SUM(a.count),0)").Joins("JOIN club_payments p ON p.id = a.payment_id").Where("a.card_id = ? AND p.void = ?", cardID, false).Scan(&n).Error
	return n, err
}
func explain(err error) (int, string) {
	var p *problem
	if errors.As(err, &p) {
		return p.status, p.message
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 404, "Record not found"
	}
	return 500, fmt.Sprint("Could not save membership record")
}

func earliestDuesPayment(month, joined string) string {
	earliest := joined[:7] + "-01"
	if month+"-01" < earliest {
		earliest = month + "-01"
	}
	return earliest
}
