package router

import "github.com/gin-gonic/gin"

func ConnectRouter(s *Server, adminToken string) *gin.Engine {
	router := gin.Default()
	api := router.Group("/api")

	api.GET("/health", s.Health)
	api.GET("/ready", s.Ready)

	// TEAMS
	teams := router.Group("/api/teams")
	teams.GET("", s.GetTeam)
	teams.POST("", s.requireAdmin(adminToken), s.CreateTeam)

	//PLAYERS
	players := router.Group("/api/teams/:tag/players")
	players.Use(s.verifyTeam)
	players.GET("", s.GetPlayer)
	players.POST("", s.requireAdmin(adminToken), s.CreatePlayer)
	return router
}
