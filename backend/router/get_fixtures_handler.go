package router

import (
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) GetFixtures(c *gin.Context) {
	result := make([]models.Fixture, 0)

	err := s.GetDB().WithContext(c.Request.Context()).
		Preload("HomeTeam").
		Preload("AwayTeam").
		Order("matchday ASC").
		Order("home_team_tag ASC").
		Find(&result).Error

	if err != nil {
		log.Printf("List fixtures failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load fixtures",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models.ToFixturesDTO(result)})
}
