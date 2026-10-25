package membership

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestAnnualRolloverAndAdvancePayment(t *testing.T) {
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
	m := Member{Name: "Annual " + uuid.NewString(), JoinedOn: "2026-10-01", Active: true}
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
	var december Due
	if err = db.First(&december, "member_id = ? AND month = ?", m.ID, "2026-12").Error; err != nil {
		t.Fatal(err)
	}
	run(Input{Type: "pay_due", MemberID: m.ID, RecordID: december.ID, PaidOn: today()})
	if err = execute(db, Input{RequestID: uuid.NewString(), Type: "set_annual_fee", Year: 2026, Amount: 150000}); err == nil {
		t.Fatal("issued annual rate changed")
	}
	run(Input{Type: "set_annual_fee", Year: 2027, Amount: 150000})
	var n int64
	db.Model(&Due{}).Where("member_id = ? AND month LIKE '2027-%'", m.ID).Count(&n)
	if n != 0 {
		t.Fatal("future year issued early")
	}
	if err = db.Transaction(func(tx *gorm.DB) error { return ensureDuesTx(tx, "2027-01-01") }); err != nil {
		t.Fatal(err)
	}
	var dues []Due
	db.Where("member_id = ? AND month LIKE '2027-%'", m.ID).Find(&dues)
	if len(dues) != 12 {
		t.Fatalf("expected full year: %d", len(dues))
	}
	for _, d := range dues {
		if d.Amount != 150000 {
			t.Fatal("wrong annual rate")
		}
	}
	if err = EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	db.First(&december, "id = ?", december.ID)
	if december.Status != "paid" || december.Amount != 100000 {
		t.Fatal("existing payment changed")
	}
}
