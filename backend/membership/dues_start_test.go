package membership

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestFirstPaymentDuesWaivers(t *testing.T) {
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
	m := Member{Name: "First payment " + uuid.NewString(), JoinedOn: "2026-06-01", Active: true}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	run := func(in Input) {
		t.Helper()
		in.RequestID = uuid.NewString()
		if err := execute(db, in); err != nil {
			t.Fatal(err)
		}
	}
	if err = EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	due := func(month string) Due {
		t.Helper()
		var d Due
		if err := db.First(&d, "member_id = ? AND month = ?", m.ID, "2026-"+month).Error; err != nil {
			t.Fatal(err)
		}
		return d
	}
	if d := due("01"); d.Status != "unpaid" {
		t.Fatal("no payment should owe January")
	}
	feb := due("02")
	run(Input{Type: "waive_due", MemberID: m.ID, RecordID: feb.ID, Revision: feb.Revision, Note: "Manual waiver"})
	june := due("06")
	run(Input{Type: "pay_due", MemberID: m.ID, RecordID: june.ID, Revision: june.Revision, PaidOn: "2026-09-01"})
	for _, month := range []string{"01", "03", "04", "05"} {
		d := due(month)
		if d.Status != "waived" || !d.AutoWaived {
			t.Fatal("earlier unpaid month not waived", month)
		}
	}
	if d := due("07"); d.Status != "unpaid" {
		t.Fatal("used receipt date instead of dues month")
	}
	if d := due("02"); d.AutoWaived {
		t.Fatal("manual waiver overwritten")
	}
	var before int64
	db.Model(&CommandRecord{}).Where("member_id = ? AND type = 'dues_start_policy'", m.ID).Count(&before)
	if err = EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	var after int64
	db.Model(&CommandRecord{}).Where("member_id = ? AND type = 'dues_start_policy'", m.ID).Count(&after)
	if before != 1 || after != before {
		t.Fatal("waiver reconciliation not idempotent")
	}
	var payment Payment
	db.First(&payment, "due_id = ? AND void=false", june.ID)
	run(Input{Type: "void_payment", MemberID: m.ID, RecordID: payment.ID, Note: "Wrong payment"})
	if d := due("01"); d.Status != "unpaid" || d.AutoWaived {
		t.Fatal("automatic waiver did not reopen after reversal")
	}
	if d := due("02"); d.Status != "waived" || d.AutoWaived {
		t.Fatal("manual waiver lost")
	}
	july := due("07")
	run(Input{Type: "pay_due", MemberID: m.ID, RecordID: july.ID, Revision: july.Revision, PaidOn: "2026-07-01"})
	if d := due("06"); d.Status != "waived" || !d.AutoWaived {
		t.Fatal("new first payment did not reconcile")
	}
	db.First(&payment, "id = ?", payment.ID)
	if !payment.Void || payment.Amount != 100000 {
		t.Fatal("original receipt lost")
	}
}
