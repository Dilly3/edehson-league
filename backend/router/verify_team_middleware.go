package router

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) verifyTeam(c *gin.Context) {
	teamTag := c.Param("tag")

	var team models.Team

	err := s.db.WithContext(c.Request.Context()).
		First(&team, "tag = ?", strings.ToUpper(teamTag)).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"error": "Team not found",
		})
		return
	}

	if err != nil {
		log.Printf("Find team failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Could not load team",
		})
		return
	}

	c.Set("teamTag", team.Tag)
	c.Set("teamID", team.ID)
	c.Next()
}
