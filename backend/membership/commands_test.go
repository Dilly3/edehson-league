package membership

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestInputHelpers(t *testing.T) {
	if !validMonth("2026-01") || validMonth("2026-13") || validMonth("2026-1") {
		t.Fatal("invalid month validation")
	}
	if validDate("2026-02-30") || !validDate("2026-02-28") {
		t.Fatal("invalid date validation")
	}
	if nextMonth("2026-12") != "2027-01-01" {
		t.Fatal("year rollover failed")
	}
}

// Use a disposable PostgreSQL database, never the app database.
// MEMBERSHIP_TEST_DSN must contain only test data: this test migrates its tables.
func TestMembershipLedgerPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("set MEMBERSHIP_TEST_DSN to a disposable PostgreSQL database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	run := func(in Input) {
		t.Helper()
		if in.RequestID == "" {
			in.RequestID = uuid.NewString()
		}
		if err := execute(db, in); err != nil {
			t.Fatalf("%s: %v", in.Type, err)
		}
	}
	run(Input{Type: "add_member", Name: "Ledger test " + uuid.NewString(), JoinedOn: today()[:7] + "-01"})
	var m Member
	if err := db.Order("created_at DESC").First(&m).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	var dues []Due
	if err := db.Where("member_id = ?", m.ID).Find(&dues).Error; err != nil {
		t.Fatal(err)
	}
	if len(dues) != 12 || dues[0].Amount != 100000 {
		t.Fatal("duplicate or incorrect dues")
	}
	d := dues[0]
	run(Input{Type: "waive_due", MemberID: m.ID, RecordID: d.ID, Revision: d.Revision, Note: "Committee waiver"})
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "pay_due", MemberID: m.ID, RecordID: d.ID, Revision: d.Revision, PaidOn: today()}); err == nil {
		t.Fatal("stale revision accepted")
	}
	run(Input{Type: "reopen_due", MemberID: m.ID, RecordID: d.ID, Revision: 1, Note: "Waiver entered in error"})
	duePayment := Input{RequestID: uuid.NewString(), Type: "pay_due", MemberID: m.ID, RecordID: d.ID, Revision: 2, PaidOn: today()}
	run(duePayment)
	run(duePayment)
	var duePayments int64
	db.Model(&Payment{}).Where("due_id = ?", d.ID).Count(&duePayments)
	if duePayments != 1 {
		t.Fatal("duplicate dues payment")
	}
	run(Input{Type: "add_cards", MemberID: m.ID, Color: "yellow", Count: 10, OccurredOn: today()})
	var card Card
	db.Where("member_id = ?", m.ID).First(&card)
	payment := Input{RequestID: uuid.NewString(), Type: "pay_cards", MemberID: m.ID, Color: "yellow", Count: 5, ExpectedAmount: 250000, PaidOn: today()}
	run(payment)
	run(payment)
	n, err := paidCount(db, card.ID)
	if err != nil || n != 5 {
		t.Fatalf("expected 5 paid: %d %v", n, err)
	}
	var p Payment
	db.Where("member_id = ? AND kind = ?", m.ID, "yellow").First(&p)
	if p.Amount != 250000 {
		t.Fatal("payment does not match five cards")
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "void_card", MemberID: m.ID, RecordID: card.ID, Note: "Correction"}); err == nil {
		t.Fatal("voided cards with active payments")
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "pay_cards", MemberID: m.ID, Color: "yellow", Count: 6, PaidOn: today()}); err == nil {
		t.Fatal("overpayment accepted")
	}
	changed := payment
	changed.Count = 1
	if err := execute(db, changed); err == nil {
		t.Fatal("reused request ID for a different payment")
	}
	// Two independent payments racing for the same remaining five cards: only one succeeds.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- execute(db, Input{RequestID: uuid.NewString(), Type: "pay_cards", MemberID: m.ID, Color: "yellow", Count: 5, ExpectedAmount: 250000, PaidOn: today()})
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("concurrent settlement overpaid")
	}
	run(Input{Type: "void_payment", MemberID: m.ID, RecordID: p.ID, Note: "Payment entry corrected"})
	n, err = paidCount(db, card.ID)
	if err != nil || n != 5 {
		t.Fatal("payment reversal did not restore outstanding cards")
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register(router.Group("/api"), db, func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) })
	public := httptest.NewRecorder()
	router.ServeHTTP(public, httptest.NewRequest("GET", "/api/membership?year="+today()[:4], nil))
	if public.Code != http.StatusOK {
		t.Fatalf("public read: %s", public.Body.String())
	}
	var response struct {
		Data RegisterData `json:"data"`
	}
	if err := json.Unmarshal(public.Body.Bytes(), &response); err != nil || len(response.Data.Members) == 0 {
		t.Fatal("invalid register response")
	}
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest("POST", "/api/membership/commands", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatal("public writes were not blocked")
	}
	history := httptest.NewRecorder()
	router.ServeHTTP(history, httptest.NewRequest("GET", "/api/membership/members/"+m.ID+"/history", nil))
	if history.Code != http.StatusOK {
		t.Fatal("member history unavailable")
	}
	// Rate changes do not rewrite issued charges or old card batches.
	run(Input{Type: "set_fee", Kind: "yellow", Amount: 75000, EffectiveFrom: today()})
	db.First(&card, "id = ?", card.ID)
	if card.UnitAmount != 50000 {
		t.Fatal("old card fee was rewritten")
	}
	run(Input{Type: "add_cards", MemberID: m.ID, Color: "yellow", Count: 1, OccurredOn: today()})
	var newest Card
	db.Where("member_id = ?", m.ID).Order("created_at DESC").First(&newest)
	if newest.UnitAmount != 75000 {
		t.Fatal("new fee not applied")
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "set_fee", Kind: "red", Amount: 200000, EffectiveFrom: "2020-01-01"}); err == nil {
		t.Fatal("backdated fee allowed")
	}
}

func TestAutomaticDuesPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL database required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	members := []Member{{Name: "Before register", JoinedOn: "2025-06-01", Active: true}, {Name: "October joiner", JoinedOn: "2026-10-04", Active: true}, {Name: "November joiner", JoinedOn: "2026-11-01", Active: true}}
	for i := range members {
		if err := db.Create(&members[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	accrue := func(date string) {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error { return ensureDuesTx(tx, date) }); err != nil {
			t.Fatal(err)
		}
	}
	accrue("2026-10-23")
	accrue("2026-10-23")
	for i, want := range []int64{12, 12, 0} {
		var n int64
		db.Model(&Due{}).Where("member_id = ?", members[i].ID).Count(&n)
		if n != want {
			t.Fatalf("member %d: expected %d charges got %d", i, want, n)
		}
	}
	var before int64
	db.Model(&Due{}).Where("month < ?", RegisterStart).Count(&before)
	if before != 0 {
		t.Fatal("charges created before 2026")
	}
	var october Due
	db.Where("member_id = ? AND month = ?", members[1].ID, "2026-10").First(&october)
	if october.Amount != 100000 {
		t.Fatal("October default must be ₦1000")
	}
	db.Model(&october).Updates(map[string]interface{}{"status": "waived", "revision": 1})
	if err := db.Create(&Fee{Kind: "monthly", EffectiveFrom: "2026-11-01", Amount: 150000}).Error; err != nil {
		t.Fatal(err)
	}
	accrue("2026-11-02")
	var november Due
	db.Where("member_id = ? AND month = ?", members[1].ID, "2026-11").First(&november)
	if november.Amount != 100000 {
		t.Fatal("legacy monthly fee changed the annual rate")
	}
	db.First(&october, "id = ?", october.ID)
	if october.Status != "waived" || october.Amount != 100000 {
		t.Fatal("accrual overwrote an existing waiver or fee")
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "set_fee", Kind: "monthly", EffectiveFrom: today()[:7] + "-01", Amount: 200000}); err == nil {
		t.Fatal("current month fee change accepted")
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "set_fee", Kind: "monthly", EffectiveFrom: nextMonth(today()[:7])[:8] + "15", Amount: 200000}); err == nil {
		t.Fatal("mid-month fee change accepted")
	}
}

func TestDuesPaidOnFirstOfJoiningMonthPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL database required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "add_member", Name: "First-day payment " + uuid.NewString(), JoinedOn: today()}); err != nil {
		t.Fatal(err)
	}
	var m Member
	db.Order("created_at DESC").First(&m)
	var due Due
	if err := db.Where("member_id = ? AND month = ?", m.ID, today()[:7]).First(&due).Error; err != nil {
		t.Fatal(err)
	}
	if err := execute(db, Input{RequestID: uuid.NewString(), Type: "pay_due", MemberID: m.ID, RecordID: due.ID, Revision: due.Revision, PaidOn: due.Month + "-01"}); err != nil {
		t.Fatal(err)
	}
	var payment Payment
	if err := db.Where("due_id = ?", due.ID).First(&payment).Error; err != nil {
		t.Fatal(err)
	}
	if payment.PaidOn != due.Month+"-01" {
		t.Fatal("incorrect payment date")
	}
}
