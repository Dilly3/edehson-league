package router

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Server) EditFixture(c *gin.Context) {
	fixtureID, err := uuid.Parse(c.Param("fixtureID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid fixture ID",
		})
		return
	}

	// A map lets us distinguish an omitted field from explicit null.
	var input map[string]json.RawMessage

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	badInput := func() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide kickoff_at as an RFC3339 timestamp or null",
		})
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		badInput()
		return
	}

	rawKickoff, exists := input["kickoff_at"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide the kickoff_at field",
		})
		return
	}
	var kickoff *time.Time
	if err := json.Unmarshal(rawKickoff, &kickoff); err != nil {
		badInput()
		return
	}

	moveMatchday := false
	expectedKeys := 1
	if raw, ok := input["move_matchday"]; ok {
		expectedKeys++
		if err := json.Unmarshal(raw, &moveMatchday); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "move_matchday must be true or false",
			})
			return
		}
	}
	if len(input) != expectedKeys {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide only kickoff_at and optionally move_matchday",
		})
		return
	}

	if kickoff != nil {
		utc := kickoff.UTC()
		kickoff = &utc
	}

	var updated models.Fixture
	errFinalized := errors.New("finalized fixture cannot be rescheduled")
	var shifted []models.Fixture

	err = s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			// Serialize date validation and writes, including concurrent admin requests.
			if err := tx.Exec("LOCK TABLE fixtures IN EXCLUSIVE MODE").Error; err != nil {
				return err
			}
			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", fixtureID.String()).
				First(&updated).Error; err != nil {
				return err
			}

			if updated.Status == "finalized" || updated.Status == "live" || updated.Status == "half_time" {
				return errFinalized
			}

			var others []models.Fixture
			if kickoff != nil || moveMatchday {
				if err := tx.Where("id <> ?", updated.ID).Find(&others).Error; err != nil {
					return err
				}
				if moveMatchday {
					shifted = shiftMatchday(updated, kickoff, others)
				}
				if err := validateFixtureDate(updated, kickoff, others); err != nil {
					return err
				}
			}

			for _, sibling := range shifted {
				if err := tx.Model(&models.Fixture{}).
					Where("id = ?", sibling.ID).
					Updates(map[string]interface{}{
						"kickoff_at": sibling.KickoffAt,
						"revision":   gorm.Expr("revision + 1"),
					}).Error; err != nil {
					return err
				}
			}

			if err := tx.Model(&updated).
				Updates(map[string]interface{}{
					"kickoff_at": kickoff,
					"revision":   gorm.Expr("revision + 1"),
				}).Error; err != nil {
				return err
			}

			return tx.
				Preload("HomeTeam").
				Preload("AwayTeam").
				First(&updated, "id = ?", fixtureID.String()).Error
		},
	)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fixture not found",
		})
		return
	}

	if errors.Is(err, errFinalized) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Live, half-time and finalized fixtures cannot be rescheduled",
		})
		return
	}

	if errors.Is(err, errFixtureDateConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	if err != nil {
		log.Printf("Update kickoff failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update kickoff time",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// shiftMatchday moves unplayed games with a kickoff in the same matchday onto
// the new WAT date, keeping each game's own time of day, or clears them when kickoff is nil. It updates others in place
// and returns the changed fixtures.
func shiftMatchday(fixture models.Fixture, kickoff *time.Time, others []models.Fixture) []models.Fixture {
	var changed []models.Fixture
	for i := range others {
		o := &others[i]
		if o.Matchday != fixture.Matchday || o.KickoffAt == nil {
			continue
		}
		if o.Status != "scheduled" && o.Status != "postponed" {
			continue
		}
		if kickoff == nil {
			o.KickoffAt = nil
			changed = append(changed, *o)
			continue
		}
		target := kickoff.In(leagueTimezone)
		local := o.KickoffAt.In(leagueTimezone)
		moved := time.Date(target.Year(), target.Month(), target.Day(),
			local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), leagueTimezone).UTC()
		o.KickoffAt = &moved
		changed = append(changed, *o)
	}
	return changed
}
