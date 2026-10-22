package router

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminSessionRequiresCorrectKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	token := strings.Repeat("k", 32)
	s.admintoken = &token
	r := ConnectRouter(s)
	for _, tc := range []struct {
		key    string
		status int
	}{
		{"", http.StatusUnauthorized},
		{"Bearer wrong", http.StatusUnauthorized},
		{"Bearer " + token, http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
		if tc.key != "" {
			req.Header.Set("Authorization", tc.key)
		}
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("expected %d, got %d", tc.status, w.Code)
		}
		if tc.status == http.StatusOK {
			if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"authenticated":true`) {
				t.Fatal("invalid auth response")
			}
		}
	}
}
