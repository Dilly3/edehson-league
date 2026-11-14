package router

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// noStoreAPI stops browsers and proxies caching API responses
// (league data and admin responses must always be fresh).
func noStoreAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}
