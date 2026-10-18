package router

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) CreateTeam(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
		Tag  string `json:"tag" binding:"required"`
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide a JSON body containing a team name",
		})
		return
	}

	tag := strings.ToUpper(strings.TrimSpace(input.Tag))

	if !regexp.MustCompile(`^EDL[0-9]{3}$`).MatchString(tag) || tag == "EDL000" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Team tag must be EDL001 through EDL999",
		})
		return
	}

	name := strings.TrimSpace(input.Name)
	length := utf8.RuneCountInString(name)

	if length < 2 || length > 20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Team name must contain 2–20 characters",
		})
		return
	}

	team := models.Team{Name: name, Tag: strings.ToUpper(tag)}

	err := s.GetDB().WithContext(c.Request.Context()).
		Create(&team).Error

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "A team with this name already exists",
		})
		return
	}

	if err != nil {
		log.Printf("Create team failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create team",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": team})

}
