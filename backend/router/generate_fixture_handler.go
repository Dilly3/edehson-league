package router

import (
	"errors"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	errTooFewTeams = errors.New("at least two teams are required")
)

func (s *Server) GenerateFixture(c *gin.Context) {
	created := 0

	err := s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			if err := tx.Exec(
				"LOCK TABLE teams IN SHARE MODE",
			).Error; err != nil {
				return err
			}

			// Serialize schedule updates and fixture writes.
			if err := tx.Exec(
				"LOCK TABLE fixtures IN EXCLUSIVE MODE",
			).Error; err != nil {
				return err
			}

			var teams []models.Team
			if err := tx.Order("tag ASC").
				Find(&teams).Error; err != nil {
				return err
			}

			if len(teams) < 2 {
				return errTooFewTeams
			}

			tags := make([]string, 0, len(teams))
			for _, team := range teams {
				tags = append(tags, team.Tag)
			}

			var existing []models.Fixture

			// Include soft-deleted fixtures to avoid duplicating
			// historical pairings or violating the unique index.
			if err := tx.Unscoped().
				Find(&existing).Error; err != nil {
				return err
			}

			additions, err := missingFixtures(tags, existing)
			if err != nil {
				return err
			}
			if len(additions) == 0 {
				return nil
			}

			if err := tx.Create(&additions).Error; err != nil {
				return err
			}

			created = len(additions)
			return nil
		},
	)

	if errors.Is(err, errTooFewTeams) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Add at least two teams before generating fixtures",
		})
		return
	}

	if errors.Is(err, errScheduleConflict) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot complete the schedule while preserving existing fixtures. The league supports up to four teams, with each team playing once per matchday.",
		})
		return
	}

	if err != nil {
		log.Printf("Update schedule failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update schedule",
		})
		return
	}

	status := http.StatusOK
	message := "Schedule already up to date"

	if created > 0 {
		status = http.StatusCreated
		message = "Missing fixtures added"
	}

	c.JSON(status, gin.H{
		"message":          message,
		"fixtures_created": created,
	})
}
func buildSchedule(tags []string) []models.Fixture {
	rotation := append([]string(nil), tags...)

	// An empty slot represents a bye for an odd number of teams.
	if len(rotation)%2 != 0 {
		rotation = append(rotation, "")
	}

	rounds := len(rotation) - 1
	result := make([]models.Fixture, 0, len(tags)*(len(tags)-1))

	for round := range rounds {
		for i := 0; i < len(rotation)/2; i++ {
			home := rotation[i]
			away := rotation[len(rotation)-1-i]

			if home == "" || away == "" {
				continue
			}

			if (round+i)%2 != 0 {
				home, away = away, home
			}

			result = append(result,
				models.Fixture{
					HomeTeamTag: home,
					AwayTeamTag: away,
					Matchday:    round + 1,
					Status:      "scheduled",
				},
				models.Fixture{
					HomeTeamTag: away,
					AwayTeamTag: home,
					Matchday:    round + 1 + rounds,
					Status:      "scheduled",
				},
			)
		}

		// Keep the first team fixed and rotate the remaining teams.
		last := rotation[len(rotation)-1]
		copy(rotation[2:], rotation[1:len(rotation)-1])
		rotation[1] = last
	}

	return result
}
