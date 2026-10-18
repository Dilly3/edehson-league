package router

import (
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) GetTeam(c *gin.Context) {
	result := make([]models.Team, 0)

	err := s.GetDB().WithContext(c.Request.Context()).
		Order("name ASC").
		Find(&result).Error
	if err != nil {
		log.Printf("List teams failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load teams",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
