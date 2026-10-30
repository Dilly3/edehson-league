package router

import (
	"github.com/dilly3/edehson-league/membership"
	"github.com/dilly3/edehson-league/vieweraccess"
	"github.com/gin-gonic/gin"
)

func ConnectRouter(s *Server) *gin.Engine {
	router := gin.Default()
	router.Use(vieweraccess.Middleware(s.GetDB(), s.GetAdminToken()))
	api := router.Group("/api")
	vieweraccess.Register(api, s.GetDB(), s.requireAdmin(s.GetAdminToken()))
	membership.Register(api, s.GetDB(), s.requireAdmin(s.GetAdminToken()))

	api.GET("/health", s.Health)
	api.GET("/ready", s.Ready)
	api.GET("/admin/session", s.requireAdmin(s.GetAdminToken()), s.AdminSession)

	// TEAMS
	teams := router.Group("/api/teams")
	teams.GET("", s.GetTeam)
	teams.PATCH("/:tag/name", s.requireAdmin(s.GetAdminToken()), s.EditTeamName)
	teams.POST("", s.requireAdmin(s.GetAdminToken()), s.CreateTeam)

	//PLAYERS
	players := router.Group("/api/teams/:tag/players")
	players.Use(s.verifyTeam)
	players.GET("", s.GetPlayer)
	players.POST("", s.requireAdmin(s.GetAdminToken()), s.CreatePlayer)
	players.PUT(
		"/:playerID",
		s.requireAdmin(s.GetAdminToken()),
		s.EditPlayer,
	)

	players.PUT(
		"/:playerID/captain",
		s.requireAdmin(s.GetAdminToken()),
		s.SetCaptain,
	)

	players.DELETE(
		"/:playerID",
		s.requireAdmin(s.GetAdminToken()),
		s.DeletePlayer,
	)

	// FIXTURES
	fixtures := router.Group("/api/fixtures")
	fixtures.GET("", s.GetFixtures)
	fixtures.GET("/:fixtureID", s.GetFixture)
	fixtures.POST("/generate", s.requireAdmin(s.GetAdminToken()), s.GenerateFixture)
	fixtures.PATCH("/:fixtureID/kickoff", s.requireAdmin(s.GetAdminToken()), s.EditFixture)
	fixtures.PATCH(
		"/:fixtureID/venue",
		s.requireAdmin(s.GetAdminToken()),
		s.EditFixtureVenue,
	)

	fixtures.PATCH("/:fixtureID/broadcast", s.requireAdmin(s.GetAdminToken()), s.EditFixtureBroadcast)

	// RESULT
	fixtures.PUT("/:fixtureID/result", s.requireAdmin(s.GetAdminToken()), s.EditResult)

	// STANDING
	api.GET("/standings", s.GetStandings)
	api.GET("/player-stats", s.GetPlayerStats)
	api.GET("/top-scorers", s.GetTopScorers)
	// RETURN ROUTER
	return router
}
