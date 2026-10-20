package router

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *Server) EditFixtureVenue(c *gin.Context) {
	fixtureID, err := uuid.Parse(c.Param("fixtureID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid fixture ID",
		})
		return
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		4096,
	)

	// Distinguish an omitted venue from an explicit null.
	var input map[string]*string

	decoder := json.NewDecoder(c.Request.Body)

	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide venue as a string or null",
		})
		return
	}

	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide exactly one JSON object",
		})
		return
	}

	venue, exists := input["venue"]
	if !exists || len(input) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide only the venue field",
		})
		return
	}

	if venue != nil {
		value := strings.TrimSpace(*venue)
		length := utf8.RuneCountInString(value)

		if length < 2 || length > 120 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Venue must contain 2–120 characters; use null to clear it",
			})
			return
		}

		venue = &value
	}

	result := s.GetDB().
		WithContext(c.Request.Context()).
		Model(&models.Fixture{}).
		Where("id = ?", fixtureID.String()).
		Updates(map[string]interface{}{
			"venue":    venue,
			"revision": gorm.Expr("revision + 1"),
		})

	if result.Error != nil {
		log.Printf("Update fixture venue failed: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update venue",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fixture not found",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
