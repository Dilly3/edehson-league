package router

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) CreatePlayer(c *gin.Context) {
	var input struct {
		Name         string `json:"name" binding:"required"`
		JerseyNumber int    `json:"jersey_number" binding:"required"`
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	name := strings.TrimSpace(input.Name)
	length := utf8.RuneCountInString(name)
	if input.JerseyNumber < 1 || input.JerseyNumber > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Chose jersey number 1-100",
		})
		return
	}
	jerseyNumber := strconv.Itoa(input.JerseyNumber)

	if length < 2 || length > 20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Player name must contain 2–20 characters",
		})
		return
	}

	player := models.Player{
		TeamID:       c.GetString("teamID"),
		Name:         name,
		JerseyNumber: jerseyNumber,
	}

	err := s.GetDB().WithContext(c.Request.Context()).
		Create(&player).Error

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This player name already exists on this team",
		})
		return
	}

	if err != nil {
		log.Printf("Create player failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create player",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": player})
}
