package router

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// AdminSession is accessible only after requireAdmin verifies the bearer key.
func (s *Server) AdminSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"authenticated": true}})
}
