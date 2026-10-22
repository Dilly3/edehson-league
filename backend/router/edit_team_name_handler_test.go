package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEditTeamNameRejectsInvalidInputBeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct{ name, tag, body string }{
		{"invalid tag", "EDL000", `{"name":"New team"}`},
		{"short name", "EDL001", `{"name":" "}`},
		{"long name", "EDL001", `{"name":"123456789012345678901"}`},
		{"tag mutation", "EDL001", `{"name":"New team","tag":"EDL002"}`},
		{"extra object", "EDL001", `{"name":"New team"} {}`},
		{"null", "EDL001", `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			s := &Server{}
			r.PATCH("/teams/:tag/name", s.EditTeamName)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/teams/"+tc.tag+"/name", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
