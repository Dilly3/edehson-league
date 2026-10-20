package router

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *Server) GetFixture(c *gin.Context) {
	fixtureID, err := uuid.Parse(c.Param("fixtureID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid fixture ID",
		})
		return
	}

	var fixture models.Fixture
	events := make([]models.MatchEvent, 0)

	err = s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			if err := tx.
				Preload("HomeTeam").
				Preload("AwayTeam").
				First(&fixture, "id = ?", fixtureID.String()).
				Error; err != nil {
				return err
			}

			return tx.
				Preload("Player").
				Where("fixture_id = ?", fixture.ID).
				Order("position ASC").
				Order("id ASC").
				Find(&events).Error
		},
		&sql.TxOptions{
			Isolation: sql.LevelRepeatableRead,
			ReadOnly:  true,
		},
	)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fixture not found",
		})
		return
	}

	if err != nil {
		log.Printf("Load fixture details failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load fixture details",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"fixture": models.ToFixtureDTO(fixture),
			"events":  models.ToMatchEventsDTO(events),
		},
	})
}
