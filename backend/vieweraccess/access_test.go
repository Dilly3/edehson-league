package vieweraccess

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPINFormat(t *testing.T) {
	for _, pin := range []string{"GM4492", "a00001", "00Z000"} {
		if !pinValid(pin) {
			t.Fatal(pin)
		}
	}
	for _, pin := range []string{"4492", "123456", "abcdef", "ABCDEF", "１２３４５６", "ab12!3", " abc12", "Ab12345", "ab123"} {
		if pinValid(pin) {
			t.Fatal(pin)
		}
	}
}
func TestPublicMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware(nil, "test-admin-key"))
	r.GET("/api/teams", func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/teams", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}

// VIEWER_ACCESS_TEST_DSN must point to a disposable PostgreSQL database.
func TestViewerAccessPostgres(t *testing.T) {
	dsn := os.Getenv("VIEWER_ACCESS_TEST_DSN")
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
	gin.SetMode(gin.TestMode)
	adminKey := "test-admin-key"
	r := gin.New()
	r.Use(Middleware(db, adminKey))
	api := r.Group("/api")
	admin := func(c *gin.Context) {
		if !adminValid(c, adminKey) {
			c.AbortWithStatus(401)
			return
		}
		c.Next()
	}
	Register(api, db, admin)
	api.GET("/teams", func(c *gin.Context) { c.JSON(200, gin.H{"data": "private"}) })
	api.GET("/health", func(c *gin.Context) { c.Status(200) })
	call := func(method, path, body string, cookie *http.Cookie, key string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		r.ServeHTTP(w, req)
		return w
	}
	var setting Setting
	db.First(&setting, 1)
	if setting.PINHash == "GM4492" || bcrypt.CompareHashAndPassword([]byte(setting.PINHash), []byte("GM4492")) != nil {
		t.Fatal("incorrect initial PIN hashing")
	}
	if w := call("GET", "/api/teams", "", nil, ""); w.Code != 401 {
		t.Fatal("API not gated", w.Code)
	}
	if w := call("GET", "/api/health", "", nil, ""); w.Code != 200 {
		t.Fatal("health blocked")
	}
	if w := call("GET", "/api/teams", "", nil, adminKey); w.Code != 200 {
		t.Fatal("admin bypass blocked")
	}
	if w := call("POST", "/api/viewer/login", `{"pin":"Ab1234"}`, nil, ""); w.Code != 401 {
		t.Fatal("wrong PIN accepted")
	}
	w := call("POST", "/api/viewer/login", `{"pin":"GM4492"}`, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api" {
		t.Fatal("unsafe cookie")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable auth")
	}
	var sessions []Session
	db.Find(&sessions)
	if len(sessions) != 1 || sessions[0].Hash == cookie.Value {
		t.Fatal("session not hashed")
	}
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 200 {
		t.Fatal("valid viewer blocked")
	}
	if w := call("PUT", "/api/admin/viewer-pin", `{"pin":"Ab0001","version":1}`, cookie, ""); w.Code != 401 {
		t.Fatal("viewer changed PIN")
	}
	if w := call("PUT", "/api/admin/viewer-pin", `{"pin":"Ab0001","version":1}`, nil, adminKey); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	db.First(&setting, 1)
	if setting.Version != 2 || bcrypt.CompareHashAndPassword([]byte(setting.PINHash), []byte("Ab0001")) != nil {
		t.Fatal("migration reset PIN")
	}
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 401 {
		t.Fatal("old session survived rotation")
	}
	if w := call("PUT", "/api/admin/viewer-pin", `{"pin":"Ab1234","version":1}`, nil, adminKey); w.Code != 409 {
		t.Fatal("stale update accepted")
	}
	if w := call("POST", "/api/viewer/login", `{"pin":"GM4492"}`, nil, ""); w.Code != 401 {
		t.Fatal("old PIN accepted")
	}
	w = call("POST", "/api/viewer/login", `{"pin":"Ab0001"}`, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie = w.Result().Cookies()[0]
	if w := call("POST", "/api/viewer/logout", "", cookie, ""); w.Code != 204 {
		t.Fatal("logout failed")
	}
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 401 {
		t.Fatal("logged-out session valid")
	}
	w = call("POST", "/api/viewer/login", `{"pin":"Ab0001"}`, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie = w.Result().Cookies()[0]
	db.Model(&Session{}).Where("hash = ?", hashToken(cookie.Value)).Update("last_active_at", time.Now().Add(-time.Hour))
	var prior Session
	db.First(&prior, "hash = ?", hashToken(cookie.Value))
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 200 {
		t.Fatal("active idle session blocked")
	}
	var polled Session
	db.First(&polled, "hash = ?", hashToken(cookie.Value))
	if !prior.LastActiveAt.Equal(polled.LastActiveAt) {
		t.Fatal("polling renewed idle session")
	}
	if w := call("POST", "/api/viewer/activity", `{}`, cookie, ""); w.Code != 204 {
		t.Fatal("activity renewal failed")
	}
	db.First(&polled, "hash = ?", hashToken(cookie.Value))
	if !polled.LastActiveAt.After(prior.LastActiveAt) {
		t.Fatal("activity did not renew")
	}
	db.Model(&Session{}).Where("hash = ?", hashToken(cookie.Value)).Update("last_active_at", time.Now().Add(-idleTimeout-time.Second))
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 401 {
		t.Fatal("idle session still valid")
	}
	if w := call("POST", "/api/viewer/activity", `{}`, cookie, ""); w.Code != 401 {
		t.Fatal("expired idle session revived")
	}
	db.Model(&Session{}).Where("hash = ?", hashToken(cookie.Value)).Update("last_active_at", time.Now())
	db.Model(&Session{}).Where("hash = ?", hashToken(cookie.Value)).Update("expires_at", time.Now().Add(-time.Hour))
	if w := call("GET", "/api/teams", "", cookie, ""); w.Code != 401 {
		t.Fatal("expired session valid")
	}
	blocked := false
	for i := 0; i < 12; i++ {
		w = call("POST", "/api/viewer/login", `{"pin":"Ab9999"}`, nil, "")
		if w.Code == 429 {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Fatal("login rate limit missing")
	}
}
