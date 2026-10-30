package router

import (
	"context"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
)

const playerStatsQuery = `
	SELECT
		p.id AS player_id,
		t.tag || '-' || p.jersey_number AS player_tag,
		p.name,
		t.tag AS team_tag,
		t.name AS team_name,
		p.jersey_number,
		COALESCE(stats.goals, 0) AS goals,
		COALESCE(stats.own_goals, 0) AS own_goals,
		COALESCE(stats.yellow_cards, 0) AS yellow_cards,
		COALESCE(stats.straight_red_cards, 0) AS straight_red_cards,
		COALESCE(stats.second_yellow_dismissals, 0)
			AS second_yellow_dismissals,
		COALESCE(stats.dismissals, 0) AS dismissals
	FROM players AS p
	JOIN teams AS t ON t.id = p.team_id
	LEFT JOIN (
		SELECT
			e.player_id,
			COUNT(*) FILTER (
				WHERE e.type = 'goal'
			) AS goals,
			COUNT(*) FILTER (
				WHERE e.type = 'own_goal'
			) AS own_goals,
			COUNT(*) FILTER (
				WHERE e.type IN ('yellow_card', 'second_yellow')
			) AS yellow_cards,
			COUNT(*) FILTER (
				WHERE e.type = 'red_card'
			) AS straight_red_cards,
			COUNT(*) FILTER (
				WHERE e.type = 'second_yellow'
			) AS second_yellow_dismissals,
			COUNT(*) FILTER (
				WHERE e.type IN ('red_card', 'second_yellow')
			) AS dismissals
		FROM match_events AS e
		JOIN fixtures AS f ON f.id = e.fixture_id
		WHERE e.deleted_at = 0
			AND f.deleted_at = 0
			AND f.status = 'finalized'
		GROUP BY e.player_id
	) AS stats ON stats.player_id = p.id
	WHERE p.deleted_at = 0
		AND t.deleted_at = 0
	ORDER BY
		goals DESC,
		LOWER(p.name) ASC,
		t.tag ASC,
		p.id ASC
`

func (s *Server) loadPlayerStats(
	ctx context.Context,
) ([]models.PlayerStatsDTO, error) {
	result := make([]models.PlayerStatsDTO, 0)

	err := s.GetDB().
		WithContext(ctx).
		Raw(playerStatsQuery).
		Scan(&result).Error

	return result, err
}

func (s *Server) GetPlayerStats(c *gin.Context) {
	result, err := s.loadPlayerStats(c.Request.Context())
	if err != nil {
		log.Printf("Load player statistics failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load player statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (s *Server) GetTopScorers(c *gin.Context) {
	stats, err := s.loadPlayerStats(c.Request.Context())
	if err != nil {
		log.Printf("Load top scorers failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load top scorers",
		})
		return
	}

	scorers := make([]models.PlayerStatsDTO, 0)

	for _, player := range stats {
		if player.Goals > 0 {
			scorers = append(scorers, player)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": scorers})
}
