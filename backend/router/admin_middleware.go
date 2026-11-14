package router

import (
	"github.com/dilly3/edehson-league/adminauth"
	"github.com/gin-gonic/gin"
)

// adminAuth lazily builds the shared authenticator (super admin key + admin sessions).
func (s *Server) adminAuth() *adminauth.Auth {
	if s.auth == nil {
		s.auth = adminauth.New(s.GetDB(), s.GetAdminToken())
	}
	return s.auth
}

// requireAdmin allows the super admin key or a signed-in admin account.
// The token argument is kept for call-site compatibility; the key comes from the server.
func (s *Server) requireAdmin(_ string) gin.HandlerFunc {
	return s.adminAuth().RequireAdmin()
}

// requireLeague guards teams, players, fixtures and results.
func (s *Server) requireLeague() gin.HandlerFunc { return s.adminAuth().RequireLeague() }

// requireRoaster guards the membership register (members, dues, cards, donations, fees).
func (s *Server) requireRoaster() gin.HandlerFunc { return s.adminAuth().RequireRoaster() }
