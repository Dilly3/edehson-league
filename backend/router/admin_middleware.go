package router

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) requireAdmin(token string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte(token))

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Admin authentication required",
			})
			return
		}

		provided := sha256.Sum256(
			[]byte(strings.TrimPrefix(header, "Bearer ")),
		)

		if subtle.ConstantTimeCompare(provided[:], expected[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid admin key",
			})
			return
		}

		c.Next()
	}
}
