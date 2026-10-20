package router

import (
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
	var input map[string]*time.Time

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide kickoff_at as an RFC3339 timestamp or null",
		})
		return
	}

	kickoff, exists := input["kickoff_at"]
	if !exists || len(input) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide only the kickoff_at field",
		})
		return
	}

	if kickoff != nil {
		utc := kickoff.UTC()
		kickoff = &utc
	}

	var updated models.Fixture
	errFinalized := errors.New("finalized fixture cannot be rescheduled")

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
			if kickoff != nil {
				if err := tx.Where("id <> ?", updated.ID).Find(&others).Error; err != nil {
					return err
				}
				if err := validateFixtureDate(updated, kickoff, others); err != nil {
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
