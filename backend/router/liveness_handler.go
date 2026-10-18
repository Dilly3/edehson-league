package router

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	if err := h.GetSQLDB().PingContext(ctx); err != nil {
		log.Printf("Database readiness check failed: %v", err)

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unavailable",
			"database": "disconnected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "connected",
	})
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "RealEdehsons League API",
	})
}
