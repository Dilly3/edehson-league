package adminauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testRouter(key string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := New(nil, key)
	a.Register(r.Group("/api"))
	r.GET("/api/protected", a.RequireAdmin(), func(c *gin.Context) { c.Status(204) })
	r.GET("/api/super", a.RequireSuperAdmin(), func(c *gin.Context) { c.Status(204) })
	r.GET("/api/league", a.RequireLeague(), func(c *gin.Context) { c.Status(204) })
	r.GET("/api/roaster", a.RequireRoaster(), func(c *gin.Context) { c.Status(204) })
	return r
}

func do(r http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestSuperAdminKey(t *testing.T) {
	key := strings.Repeat("k", 32)
	r := testRouter(key)
	w := do(r, "GET", "/api/admin/session", "Bearer "+key)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"super_admin"`) {
		t.Fatalf("super key session: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/protected", "/api/super", "/api/league", "/api/roaster"} {
		if w := do(r, "GET", path, "Bearer "+key); w.Code != 204 {
			t.Fatalf("%s with super key: %d", path, w.Code)
		}
	}
}

func TestRejectsMissingAndWrongCredentials(t *testing.T) {
	r := testRouter(strings.Repeat("k", 32))
	for _, auth := range []string{"", "Bearer wrong", "Bearer " + strings.Repeat("a", 64), "Basic x"} {
		for _, path := range []string{"/api/admin/session", "/api/protected", "/api/super", "/api/admin/users"} {
			if w := do(r, "GET", path, auth); w.Code != 401 {
				t.Fatalf("%q on %s: expected 401, got %d", auth, path, w.Code)
			}
		}
	}
}

func TestValidators(t *testing.T) {
	for in, want := range map[string]bool{"ada": true, "ada.okafor_2": true, "ab": false, "Ada": false, "a b c": false, strings.Repeat("a", 33): false} {
		if usernameValid(in) != want {
			t.Errorf("usernameValid(%q) != %v", in, want)
		}
	}
	for in, want := range map[string]bool{"password1": true, "short1": false, "allletters": false, "12345678": false, strings.Repeat("a", 72) + "1": false} {
		if passwordValid(in) != want {
			t.Errorf("passwordValid(%q) != %v", in, want)
		}
	}
	if normalizeUsername("  Ada.Okafor ") != "ada.okafor" {
		t.Fatal("normalizeUsername")
	}
}

func TestPasswordsAreCaseInsensitive(t *testing.T) {
	if foldPassword("Secret123") != foldPassword("sECRET123") {
		t.Fatal("passwords differing only in case must fold to the same value")
	}
}

func TestPermissions(t *testing.T) {
	leagueOnly := Principal{Role: RoleAdmin, CanLeague: true}
	if !leagueOnly.Can(PermLeague) || leagueOnly.Can(PermRoaster) || !leagueOnly.Can("") {
		t.Fatal("league-only admin must reach league routes but not roaster routes")
	}
	roasterOnly := Principal{Role: RoleAdmin, CanRoaster: true}
	if roasterOnly.Can(PermLeague) || !roasterOnly.Can(PermRoaster) {
		t.Fatal("roaster-only admin must reach roaster routes but not league routes")
	}
	if (Principal{}).Can("unknown") {
		t.Fatal("unknown permissions must be denied")
	}
}

func TestSuperAdminSessionReportsAllPermissions(t *testing.T) {
	key := strings.Repeat("k", 32)
	w := do(testRouter(key), "GET", "/api/admin/session", "Bearer "+key)
	if !strings.Contains(w.Body.String(), `"permissions":{"league":true,"roaster":true}`) {
		t.Fatalf("unexpected session body: %s", w.Body.String())
	}
}

// Regression: creating a League-only admin must store can_roaster = false,
// not let the column default (true) take over. Checks the generated SQL only.
func TestInsertUserWritesFalsePermissions(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=x dbname=x sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	user := User{ID: "00000000-0000-0000-0000-000000000001", Username: "ada", PasswordHash: "h", Active: true, CanLeague: true, CanRoaster: false}
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		_, _ = insertUser(tx, &user)
		return tx
	})
	if !strings.Contains(sql, `"can_roaster"`) || !strings.Contains(sql, "false") {
		t.Fatalf("insert must write can_roaster=false explicitly: %s", sql)
	}
}
