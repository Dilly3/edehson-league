package membership

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestEditDuesPaymentDate(t *testing.T) {
	dsn := os.Getenv("MEMBERSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	m := Member{Name: "Date edit " + uuid.NewString(), JoinedOn: "2026-01-01", Active: true}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	if err = EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	var d Due
	db.First(&d, "member_id = ? AND month = ?", m.ID, "2026-12")
	run := func(in Input) error { in.RequestID = uuid.NewString(); return execute(db, in) }
	if err = run(Input{Type: "pay_due", MemberID: m.ID, RecordID: d.ID, PaidOn: "2026-01-01", Note: "Original receipt"}); err != nil {
		t.Fatal(err)
	}
	var p Payment
	db.First(&p, "due_id = ? AND void = false", d.ID)
	old, note := p.PaidOn, p.Note
	in := Input{Type: "edit_dues_payment", MemberID: m.ID, RecordID: p.ID, ExpectedPaidOn: &old, ExpectedNote: &note, PaidOn: "2026-02-01", Note: "Date correction"}
	if err = run(in); err != nil {
		t.Fatal(err)
	}
	if err = run(in); err == nil {
		t.Fatal("stale edit accepted")
	}
	var changed Payment
	db.First(&changed, "id = ?", p.ID)
	db.First(&d, "id = ?", d.ID)
	if changed.PaidOn != "2026-02-01" || changed.Amount != p.Amount || changed.Note != note || changed.Void || d.Status != "paid" {
		t.Fatal("date correction changed ledger")
	}
	now := changed.PaidOn
	in.ExpectedPaidOn = &now
	in.PaidOn = "2027-01-01"
	if err = run(in); err == nil {
		t.Fatal("future receipt accepted")
	}
	in.PaidOn = "2025-12-31"
	if err = run(in); err == nil {
		t.Fatal("pre-joining receipt accepted")
	}
	if err = run(Input{Type: "void_payment", MemberID: m.ID, RecordID: p.ID, Note: "Error"}); err != nil {
		t.Fatal(err)
	}
	in.PaidOn = "2026-03-01"
	if err = run(in); err == nil {
		t.Fatal("void payment edited")
	}
	var count int64
	db.Model(&CommandRecord{}).Where("type = ? AND member_id = ?", "edit_dues_payment", m.ID).Count(&count)
	if count != 1 {
		t.Fatal("missing audit record")
	}
}
