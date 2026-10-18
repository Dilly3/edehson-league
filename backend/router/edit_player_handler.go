package router

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) EditPlayer(c *gin.Context) {
	playerID, err := uuid.Parse(c.Param("playerID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	var input struct {
		Name         string `json:"name"`
		JerseyNumber int    `json:"jersey_number"`
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide a player name and jersey number",
		})
		return
	}

	name := strings.TrimSpace(input.Name)
	length := utf8.RuneCountInString(name)

	if length < 2 || length > 20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Player name must contain 2–20 characters",
		})
		return
	}

	if input.JerseyNumber < 1 || input.JerseyNumber > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Chose jersey number 1-100",
		})
		return
	}

	jerseyNumber := strconv.Itoa(input.JerseyNumber)

	teamID := c.GetString("teamID")

	err = s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			var player models.Player

			if err := tx.
				Where("id = ? AND team_id = ?", playerID.String(), teamID).
				First(&player).Error; err != nil {
				return err
			}

			result := tx.Model(&player).Updates(map[string]interface{}{
				"name":          name,
				"jersey_number": jerseyNumber,
			})

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}

			return nil
		},
	)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Player not found on this team",
		})
		return
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This player name already exists on this team",
		})
		return
	}

	if err != nil {
		log.Printf("Update player failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update player",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
