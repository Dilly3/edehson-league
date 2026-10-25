package membership

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMemberJoiningDateEditingPostgres(t *testing.T) {
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
	run := func(in Input) {
		t.Helper()
		if in.RequestID == "" {
			in.RequestID = uuid.NewString()
		}
		if err := execute(db, in); err != nil {
			t.Fatal(in.Type, err)
		}
	}
	name := "Joining date test " + uuid.NewString()
	run(Input{Type: "add_member", Name: name, JoinedOn: "2026-03-01"})
	var m Member
	if err = db.Where("name = ?", name).First(&m).Error; err != nil {
		t.Fatal(err)
	}
	var march, april Due
	db.Where("member_id = ? AND month = ?", m.ID, "2026-03").First(&march)
	db.Where("member_id = ? AND month = ?", m.ID, "2026-04").First(&april)
	run(Input{Type: "pay_due", MemberID: m.ID, RecordID: march.ID, Revision: march.Revision, PaidOn: "2026-03-01"})
	run(Input{Type: "waive_due", MemberID: m.ID, RecordID: april.ID, Revision: april.Revision, Note: "Existing waiver"})
	var before []Due
	db.Where("member_id = ?", m.ID).Find(&before)
	run(Input{Type: "edit_member", MemberID: m.ID, Name: m.Name, Active: true, Revision: m.Revision, JoinedOn: "2026-01-05"})
	db.First(&m, "id = ?", m.ID)
	if m.JoinedOn != "2026-01-05" {
		t.Fatal("date not saved")
	}
	var dues []Due
	db.Where("member_id = ?", m.ID).Find(&dues)
	if len(dues) != len(before) {
		t.Fatal("joining date edit changed issued dues")
	}
	for _, old := range before {
		var got Due
		db.First(&got, "id = ?", old.ID)
		if got.Status != old.Status || got.Amount != old.Amount || got.Revision != old.Revision {
			t.Fatal("issued due changed")
		}
	}
	count := len(dues)
	run(Input{Type: "edit_member", MemberID: m.ID, Name: m.Name, Active: true, Revision: m.Revision, JoinedOn: "2026-05-10"})
	var n int64
	db.Model(&Due{}).Where("member_id = ?", m.ID).Count(&n)
	if int(n) != count {
		t.Fatal("later date removed dues")
	}
	db.First(&m, "id = ?", m.ID)
	for _, date := range []string{"2026-02-30", "1899-12-31", time.Now().AddDate(0, 0, 2).Format("2006-01-02")} {
		if execute(db, Input{RequestID: uuid.NewString(), Type: "edit_member", MemberID: m.ID, Name: m.Name, Active: true, Revision: m.Revision, JoinedOn: date}) == nil {
			t.Fatal("invalid date accepted", date)
		}
	}
	if execute(db, Input{RequestID: uuid.NewString(), Type: "edit_member", MemberID: m.ID, Name: m.Name, Active: true, Revision: 0, JoinedOn: "2026-01-01"}) == nil {
		t.Fatal("stale edit accepted")
	}
	run(Input{Type: "edit_member", MemberID: m.ID, Name: m.Name, Active: true, Revision: m.Revision})
	db.First(&m, "id = ?", m.ID)
	if m.JoinedOn != "2026-05-10" {
		t.Fatal("old client cleared date")
	}
	var paid Due
	db.First(&paid, "id = ?", march.ID)
	var waived Due
	db.First(&waived, "id = ?", april.ID)
	if paid.Status != "paid" || waived.Status != "waived" {
		t.Fatal("payment/waiver history lost")
	}
	var logs int64
	db.Model(&CommandRecord{}).Where("member_id = ? AND type = ? AND before LIKE ? AND after LIKE ?", m.ID, "edit_member", "%joined_on%", "%joined_on%").Count(&logs)
	if logs != 3 {
		t.Fatal("missing date audit")
	}
}
