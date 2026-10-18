package router

import "github.com/gin-gonic/gin"

func ConnectRouter(h *Handler) *gin.Engine {
	router := gin.Default()
	api := router.Group("/api")
	api.GET("/health", h.Health)
	api.GET("/ready", h.Ready)
	return router
}
