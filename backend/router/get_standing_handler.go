package router

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) GetStandings(c *gin.Context) {
	var teams []models.Team
	var fixtures []models.Fixture

	err := s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			if err := tx.Find(&teams).Error; err != nil {
				return err
			}

			return tx.
				Where("status = ?", "finalized").
				Find(&fixtures).Error
		},
		&sql.TxOptions{
			Isolation: sql.LevelRepeatableRead,
			ReadOnly:  true,
		},
	)

	if err != nil {
		log.Printf("Load standings failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load standings",
		})
		return
	}

	table, err := calculateStandings(teams, fixtures)
	if err != nil {
		log.Printf("Calculate standings failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Standings contain inconsistent match data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": table})
}
