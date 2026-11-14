package membership

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// End-to-end: player uploads one receipt covering dues + yellow + red cards,
// admin confirms, and the register shows those items paid. Uses a disposable
// database from MEMBERSHIP_TEST_DSN, like the other Postgres tests.
func TestPaymentRequestFlowPostgres(t *testing.T) {
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
	SetReceiptStore(localStore{dir: t.TempDir()})
	run := func(in Input) {
		t.Helper()
		in.RequestID = uuid.NewString()
		if err := execute(db, in); err != nil {
			t.Fatalf("%s: %v", in.Type, err)
		}
	}
	name := "Receipt Tester " + uuid.NewString()[:8]
	run(Input{Type: "add_member", Name: name, JoinedOn: today()[:7] + "-01"})
	var m Member
	if err := db.Where("name = ?", name).First(&m).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureDues(db); err != nil {
		t.Fatal(err)
	}
	run(Input{Type: "add_cards", MemberID: m.ID, Color: "yellow", Count: 3, OccurredOn: today()})
	run(Input{Type: "add_cards", MemberID: m.ID, Color: "red", Count: 1, OccurredOn: today()})
	var unpaid []Due
	db.Where("member_id = ? AND status = 'unpaid'", m.ID).Order("month").Find(&unpaid)
	if len(unpaid) < 2 {
		t.Fatalf("need two unpaid months, got %d", len(unpaid))
	}
	months := []string{unpaid[0].Month, unpaid[1].Month}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	allow := func(c *gin.Context) { c.Next() }
	Register(r.Group("/api"), db, allow, allow)

	upload := func(fields map[string]string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		for k, v := range fields {
			_ = w.WriteField(k, v)
		}
		fw, _ := w.CreateFormFile("receipt", "transfer.png")
		fw.Write([]byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 64)))
		w.Close()
		req := httptest.NewRequest("POST", "/api/membership/payment-requests", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	base := map[string]string{"member_id": m.ID, "months": strings.Join(months, ","), "yellow_count": "2", "red_count": "1", "amount_sent": "5000", "paid_on": today()}

	rec := upload(base)
	if rec.Code != 201 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	var created struct{ Data PaymentRequest }
	json.Unmarshal(rec.Body.Bytes(), &created)
	pr := created.Data
	var yCard, rCard Card
	db.Where("member_id = ? AND color = 'yellow'", m.ID).First(&yCard)
	db.Where("member_id = ? AND color = 'red'", m.ID).First(&rCard)
	wantDues, wantYellow, wantRed := unpaid[0].Amount+unpaid[1].Amount, 2*yCard.UnitAmount, rCard.UnitAmount
	if pr.DuesAmount != wantDues || pr.YellowAmt != wantYellow || pr.RedAmt != wantRed || pr.Amount != wantDues+wantYellow+wantRed || pr.AmountSent != 500000 {
		t.Fatalf("pricing wrong: %+v", pr)
	}
	if pr.MemberName != name {
		t.Fatalf("receipt not recorded against the player name: %q", pr.MemberName)
	}

	// Same month again while pending is refused; more cards than remain is refused.
	if rec := upload(map[string]string{"member_id": m.ID, "months": months[0], "amount_sent": "1000", "paid_on": today()}); rec.Code != 409 {
		t.Fatalf("duplicate month accepted: %d", rec.Code)
	}
	if rec := upload(map[string]string{"member_id": m.ID, "yellow_count": "2", "amount_sent": "1000", "paid_on": today()}); rec.Code != 409 {
		t.Fatalf("over-claimed yellow cards accepted: %d %s", rec.Code, rec.Body.String())
	}
	// Non-image upload is refused.
	{
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		for k, v := range base {
			_ = w.WriteField(k, v)
		}
		fw, _ := w.CreateFormFile("receipt", "x.html")
		fw.Write([]byte("<html><script>alert(1)</script></html>"))
		w.Close()
		req := httptest.NewRequest("POST", "/api/membership/payment-requests", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != 400 {
			t.Fatalf("html receipt accepted: %d", rr.Code)
		}
	}

	// Admin can stream the stored receipt.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/membership/payment-requests/"+pr.ID+"/receipt", nil))
	if rr.Code != 200 || rr.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("receipt download: %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}

	// Confirm records everything.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("POST", "/api/membership/payment-requests/"+pr.ID+"/confirm", nil))
	if rr.Code != 204 {
		t.Fatalf("confirm: %d %s", rr.Code, rr.Body.String())
	}
	var paidDues int64
	db.Model(&Due{}).Where("member_id = ? AND month IN ? AND status = 'paid'", m.ID, months).Count(&paidDues)
	if paidDues != 2 {
		t.Fatalf("expected 2 dues paid, got %d", paidDues)
	}
	var yellowPaid, redPaid Payment
	db.Where("member_id = ? AND kind = 'yellow'", m.ID).First(&yellowPaid)
	db.Where("member_id = ? AND kind = 'red'", m.ID).First(&redPaid)
	if yellowPaid.Amount != wantYellow || redPaid.Amount != wantRed || yellowPaid.PaidOn != today() {
		t.Fatalf("card payments wrong: yellow %+v red %+v", yellowPaid, redPaid)
	}
	var after PaymentRequest
	db.First(&after, "id = ?", pr.ID)
	if after.Status != RequestConfirmed || after.ReviewedAt == nil {
		t.Fatalf("request not marked confirmed: %+v", after)
	}
	if after.ReviewedBy != "Admin" {
		t.Fatalf("reviewer not recorded: %q", after.ReviewedBy)
	}
	var linked int64
	db.Model(&Payment{}).Where("receipt_request_id = ?", pr.ID).Count(&linked)
	if linked != 4 { // two months + one yellow payment + one red payment
		t.Fatalf("expected 4 payments linked to the receipt, got %d", linked)
	}
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("POST", "/api/membership/payment-requests/"+pr.ID+"/confirm", nil))
	if rr.Code != 409 {
		t.Fatalf("second confirm should conflict: %d", rr.Code)
	}

	// A rejected receipt records nothing.
	rec = upload(map[string]string{"member_id": m.ID, "yellow_count": "1", "amount_sent": "500", "paid_on": today()})
	if rec.Code != 201 {
		t.Fatalf("second upload: %d %s", rec.Code, rec.Body.String())
	}
	json.Unmarshal(rec.Body.Bytes(), &created)
	req := httptest.NewRequest("POST", "/api/membership/payment-requests/"+created.Data.ID+"/reject", strings.NewReader(`{"reason":"Amount not received"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("reject: %d %s", rr.Code, rr.Body.String())
	}
	var rejected PaymentRequest
	db.First(&rejected, "id = ?", created.Data.ID)
	if rejected.Status != RequestRejected || rejected.ReviewNote != "Amount not received" || rejected.ReviewedBy != "Admin" {
		t.Fatalf("rejection not recorded: %+v", rejected)
	}
	var yellowPayments int64
	db.Model(&Payment{}).Where("member_id = ? AND kind = 'yellow'", m.ID).Count(&yellowPayments)
	if yellowPayments != 1 {
		t.Fatalf("rejected receipt recorded a payment")
	}

	// Super admin deletes receipt files: pending ones are protected, records stay.
	pendingRec := upload(map[string]string{"member_id": m.ID, "yellow_count": "1", "amount_sent": "500", "paid_on": today()})
	if pendingRec.Code != 201 {
		t.Fatalf("third upload: %d %s", pendingRec.Code, pendingRec.Body.String())
	}
	var pendingReq struct{ Data PaymentRequest }
	json.Unmarshal(pendingRec.Body.Bytes(), &pendingReq)
	del := func(id string) int {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest("DELETE", "/api/membership/payment-requests/"+id+"/receipt", nil))
		return rr.Code
	}
	// Automatic retention keeps receipts reviewed within the last 90 days.
	if n, _, _, err := purgeReceiptFiles(context.Background(), db, time.Now().UTC().Add(-ReceiptRetention)); err != nil || n != 0 {
		t.Fatalf("retention deleted recent receipts: %d %v", n, err)
	}
	if code := del(pendingReq.Data.ID); code != 409 {
		t.Fatalf("pending receipt file deleted: %d", code)
	}
	if code := del(pr.ID); code != 204 {
		t.Fatalf("delete confirmed receipt file: %d", code)
	}
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/membership/payment-requests/"+pr.ID+"/receipt", nil))
	if rr.Code != 410 {
		t.Fatalf("deleted receipt should be gone: %d", rr.Code)
	}
	db.Model(&Payment{}).Where("receipt_request_id = ?", pr.ID).Count(&linked)
	if linked != 4 {
		t.Fatal("deleting the file must keep the payments")
	}
	purge := httptest.NewRequest("POST", "/api/membership/payment-requests/purge-files", strings.NewReader(`{"older_than_days":0}`))
	purge.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, purge)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"deleted":1`) {
		t.Fatalf("purge should delete only the rejected receipt's file: %d %s", rr.Code, rr.Body.String())
	}

	// Retention setting: defaults to 90 days, super admin can change it within 1–3650.
	settings := func(method, body string) (int, string) {
		req := httptest.NewRequest(method, "/api/membership/receipt-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code, rr.Body.String()
	}
	db.Exec("DELETE FROM club_receipt_settings")
	if code, body := settings("GET", ""); code != 200 || !strings.Contains(body, `"retention_days":90`) {
		t.Fatalf("default retention: %d %s", code, body)
	}
	if code, _ := settings("PUT", `{"retention_days":0}`); code != 400 {
		t.Fatalf("zero-day retention accepted: %d", code)
	}
	if code, _ := settings("PUT", `{"retention_days":30}`); code != 200 {
		t.Fatalf("save retention: %d", code)
	}
	if days, err := retentionDays(context.Background(), db); err != nil || days != 30 {
		t.Fatalf("saved retention not used: %d %v", days, err)
	}

	// Payment account: 10-digit check and public read.
	put := func(body string) int {
		req := httptest.NewRequest("PUT", "/api/membership/payment-account", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}
	if code := put(`{"bank_name":"GTBank","account_name":"Great Minds Sport","account_number":"12345"}`); code != 400 {
		t.Fatalf("short account number accepted: %d", code)
	}
	if code := put(`{"bank_name":"GTBank","account_name":"Great Minds Sport","account_number":"0123456789"}`); code != 200 {
		t.Fatalf("save account: %d", code)
	}
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/membership/payment-account", nil))
	if !strings.Contains(rr.Body.String(), `"account_number":"0123456789"`) {
		t.Fatalf("account not returned: %s", rr.Body.String())
	}
}
