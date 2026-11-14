package router

import (
	"github.com/dilly3/edehson-league/membership"
	"github.com/gin-gonic/gin"
)

func ConnectRouter(s *Server) *gin.Engine {
	router := gin.Default()
	router.Use(noStoreAPI())
	api := router.Group("/api")
	membership.Register(api, s.GetDB(), s.requireRoaster(), s.adminAuth().RequireSuperAdmin())

	api.GET("/health", s.Health)
	api.GET("/ready", s.Ready)
	// Admin sign-in, session and (super admin only) admin account management.
	s.adminAuth().Register(api)

	// TEAMS
	teams := router.Group("/api/teams")
	teams.GET("", s.GetTeam)
	teams.PATCH("/:tag/name", s.requireLeague(), s.EditTeamName)
	teams.POST("", s.requireLeague(), s.CreateTeam)

	//PLAYERS
	players := router.Group("/api/teams/:tag/players")
	players.Use(s.verifyTeam)
	players.GET("", s.GetPlayer)
	players.POST("", s.requireLeague(), s.CreatePlayer)
	players.PUT(
		"/:playerID",
		s.requireLeague(),
		s.EditPlayer,
	)

	players.PUT(
		"/:playerID/captain",
		s.requireLeague(),
		s.SetCaptain,
	)

	players.DELETE(
		"/:playerID",
		s.requireLeague(),
		s.DeletePlayer,
	)

	// FIXTURES
	fixtures := router.Group("/api/fixtures")
	fixtures.GET("", s.GetFixtures)
	fixtures.GET("/:fixtureID", s.GetFixture)
	fixtures.POST("/generate", s.requireLeague(), s.GenerateFixture)
	fixtures.PATCH("/:fixtureID/kickoff", s.requireLeague(), s.EditFixture)
	fixtures.PATCH(
		"/:fixtureID/venue",
		s.requireLeague(),
		s.EditFixtureVenue,
	)

	fixtures.PATCH("/:fixtureID/broadcast", s.requireLeague(), s.EditFixtureBroadcast)

	// RESULT
	fixtures.PUT("/:fixtureID/result", s.requireLeague(), s.EditResult)

	// STANDING
	api.GET("/standings", s.GetStandings)
	api.GET("/player-stats", s.GetPlayerStats)
	api.GET("/top-scorers", s.GetTopScorers)
	// RETURN ROUTER
	return router
}
