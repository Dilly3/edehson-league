package membership

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestDonationEditingPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	m := Member{Name: "Donation edit test", JoinedOn: "2026-01-01", Active: true}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	p := Payment{MemberID: m.ID, Kind: "donation", Amount: 1000000, Note: "Original purpose"}
	if err = db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	oldDate, oldNote := p.PaidOn, p.Note
	in := Input{RequestID: uuid.NewString(), Type: "edit_donation", MemberID: m.ID, RecordID: p.ID, PaidOn: "2026-06-06", Note: "Football", ExpectedPaidOn: &oldDate, ExpectedNote: &oldNote}
	if err = execute(db, in); err != nil {
		t.Fatal(err)
	}
	if err = execute(db, in); err != nil {
		t.Fatal("retry:", err)
	}
	var got Payment
	db.First(&got, "id = ?", p.ID)
	if got.Amount != p.Amount || got.PaidOn != in.PaidOn || got.Note != "Football" {
		t.Fatal("incorrect donation update")
	}
	var logs int64
	db.Model(&CommandRecord{}).Where("request_id = ?", in.RequestID).Count(&logs)
	if logs != 1 {
		t.Fatal("duplicate audit record")
	}
	in.RequestID = uuid.NewString()
	if execute(db, in) == nil {
		t.Fatal("stale update accepted")
	}
	newDate, newNote := got.PaidOn, got.Note
	in.ExpectedPaidOn = &newDate
	in.ExpectedNote = &newNote
	in.PaidOn = "2026-02-30"
	if execute(db, in) == nil {
		t.Fatal("invalid date accepted")
	}
	in.PaidOn = "2026-06-06"
	in.Note = " "
	if execute(db, in) == nil {
		t.Fatal("empty purpose accepted")
	}
	in.Note = "Changed"
	db.Model(&got).Update("kind", "monthly")
	if execute(db, in) == nil {
		t.Fatal("monthly payment edited")
	}
	db.Model(&got).Updates(map[string]interface{}{"kind": "donation", "void": true})
	if execute(db, in) == nil {
		t.Fatal("void donation edited")
	}
}

func TestCardPaymentDateEditingPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	m := Member{Name: "Card date test", JoinedOn: "2026-01-01", Active: true}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	for _, color := range []string{"yellow", "red"} {
		c := Card{MemberID: m.ID, Color: color, Count: 2, UnitAmount: 50000, OccurredOn: "2026-03-01"}
		if err = db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		p := Payment{MemberID: m.ID, Kind: color, Amount: 50000, Note: "Original receipt"}
		if err = db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		allocation := Allocation{PaymentID: p.ID, CardID: c.ID, Count: 1}
		if err = db.Create(&allocation).Error; err != nil {
			t.Fatal(err)
		}
		oldDate, oldNote := p.PaidOn, p.Note
		in := Input{RequestID: uuid.NewString(), Type: "edit_card_payment", MemberID: m.ID, RecordID: p.ID, PaidOn: "2026-03-02", Note: "Confirmed receipt date", ExpectedPaidOn: &oldDate, ExpectedNote: &oldNote}
		if err = execute(db, in); err != nil {
			t.Fatal(err)
		}
		if err = execute(db, in); err != nil {
			t.Fatal("retry:", err)
		}
		var got Payment
		db.First(&got, "id = ?", p.ID)
		if got.PaidOn != in.PaidOn || got.Amount != p.Amount || got.Note != p.Note || got.Kind != color {
			t.Fatal("payment fields changed incorrectly")
		}
		var a Allocation
		db.First(&a, "id = ?", allocation.ID)
		if a.Count != 1 || a.CardID != c.ID || a.PaymentID != p.ID {
			t.Fatal("allocations changed")
		}
		var logs int64
		db.Model(&CommandRecord{}).Where("request_id = ? AND before <> '' AND after <> ''", in.RequestID).Count(&logs)
		if logs != 1 {
			t.Fatal("missing/duplicate audit")
		}
		in.RequestID = uuid.NewString()
		if execute(db, in) == nil {
			t.Fatal("stale edit accepted")
		}
		in.ExpectedPaidOn = &got.PaidOn
		in.PaidOn = "2026-02-28"
		if execute(db, in) == nil {
			t.Fatal("payment before issue accepted")
		}
		in.PaidOn = "2999-01-01"
		if execute(db, in) == nil {
			t.Fatal("future date accepted")
		}
		in.PaidOn = "2026-03-02"
		in.Note = " "
		if err := execute(db, in); err != nil {
			t.Fatal("optional note rejected:", err)
		}
		in.RequestID = uuid.NewString()
		in.Note = "Correction"
		db.Model(&got).Update("kind", "monthly")
		if execute(db, in) == nil {
			t.Fatal("monthly payment edited")
		}
		db.Model(&got).Updates(map[string]interface{}{"kind": color, "void": true})
		if execute(db, in) == nil {
			t.Fatal("void payment edited")
		}
	}
}

func TestIndividualCardEditingPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	m := Member{Name: "Individual card test", JoinedOn: "2026-01-01", Active: true}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	c := Card{MemberID: m.ID, Color: "yellow", Count: 6, UnitAmount: 50000, Note: "Opening balance"}
	if err = db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	p := Payment{MemberID: m.ID, Kind: "yellow", Amount: 150000, PaidOn: "2026-04-01"}
	if err = db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	a := Allocation{PaymentID: p.ID, CardID: c.ID, Count: 3}
	if err = db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	oldDate := ""
	in := Input{RequestID: uuid.NewString(), Type: "edit_card", MemberID: m.ID, RecordID: c.ID, CardIndex: 1, ExpectedCount: 6, ExpectedOccurredOn: &oldDate, OccurredOn: "2026-04-02"}
	if execute(db, in) == nil {
		t.Fatal("card dated after settlement")
	}
	var count int64
	db.Model(&Card{}).Where("member_id = ?", m.ID).Count(&count)
	if count != 1 {
		t.Fatal("failed edit changed records")
	}
	in.OccurredOn = "2026-03-15"
	if err = execute(db, in); err != nil {
		t.Fatal(err)
	}
	if err = execute(db, in); err != nil {
		t.Fatal("retry:", err)
	}
	var cards []Card
	db.Where("member_id = ?", m.ID).Find(&cards)
	if len(cards) != 6 {
		t.Fatal("expected six individual cards")
	}
	changed := 0
	for _, card := range cards {
		if card.Count != 1 || card.UnitAmount != 50000 || card.Note != "Opening balance" {
			t.Fatal("incorrect split")
		}
		if card.OccurredOn == in.OccurredOn {
			changed++
		} else if card.OccurredOn != "" {
			t.Fatal("invented issue date")
		}
	}
	if changed != 1 {
		t.Fatal("edited more than one card")
	}
	var allocations []Allocation
	db.Where("payment_id = ?", p.ID).Find(&allocations)
	paid := 0
	for _, allocation := range allocations {
		paid += allocation.Count
		if allocation.Count != 1 {
			t.Fatal("allocation was not split")
		}
	}
	if paid != 3 || len(allocations) != 3 {
		t.Fatal("settlement totals changed")
	}
	var payment Payment
	db.First(&payment, "id = ?", p.ID)
	if payment.Amount != 150000 || payment.PaidOn != "2026-04-01" {
		t.Fatal("receipt changed")
	}
	in.RequestID = uuid.NewString()
	if execute(db, in) == nil {
		t.Fatal("stale group edit accepted")
	}
	var other Member
	other = Member{Name: "Other member", JoinedOn: "2026-01-01", Active: true}
	db.Create(&other)
	in.MemberID = other.ID
	if execute(db, in) == nil {
		t.Fatal("edited another member's card")
	}
}

func TestDefaultHistoricalCardDatesPostgres(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	m := Member{Name: "Default date test", JoinedOn: "2026-01-01", Active: true}
	db.Create(&m)
	c := Card{MemberID: m.ID, Color: "yellow", Count: 6, UnitAmount: 50000}
	db.Create(&c)
	p := Payment{MemberID: m.ID, Kind: "yellow", Amount: 100000}
	db.Create(&p)
	donation := Payment{MemberID: m.ID, Kind: "donation", Amount: 50000}
	db.Create(&donation)
	known := Card{MemberID: m.ID, Color: "red", Count: 1, UnitAmount: 100000, OccurredOn: "2026-06-01"}
	db.Create(&known)
	run := func() {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error { return defaultHistoricalCardDates(tx) }); err != nil {
			t.Fatal(err)
		}
	}
	run()
	run()
	db.First(&c, "id = ?", c.ID)
	db.First(&p, "id = ?", p.ID)
	db.First(&donation, "id = ?", donation.ID)
	db.First(&known, "id = ?", known.ID)
	if c.OccurredOn != "2026-01-01" || p.PaidOn != "2026-01-01" {
		t.Fatal("missing historical defaults")
	}
	if c.Count != 6 || p.Amount != 100000 || donation.PaidOn != "" || known.OccurredOn != "2026-06-01" {
		t.Fatal("unrelated fields changed")
	}
	var logs int64
	db.Model(&CommandRecord{}).Where("member_id = ? AND type = ?", m.ID, "default_card_dates").Count(&logs)
	if logs != 1 {
		t.Fatal("default migration not idempotent")
	}
}
