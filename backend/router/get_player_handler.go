package router

import (
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) GetPlayer(c *gin.Context) {
	result := make([]models.Player, 0)

	err := s.GetDB().WithContext(c.Request.Context()).
		Preload("Team").
		Where("team_id = ?", c.GetString("teamID")).
		Order("name ASC").
		Find(&result).Error

	if err != nil {
		log.Printf("List players failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load players",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models.ToPlayersDTO(result)})
}
