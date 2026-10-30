package router

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *Server) EditFixtureBroadcast(c *gin.Context) {
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

	// Distinguish an omitted broadcast_url from an explicit null.
	var input map[string]*string

	decoder := json.NewDecoder(c.Request.Body)

	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide broadcast_url as a string or null",
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

	broadcast_url, exists := input["broadcast_url"]
	if !exists || len(input) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide only the broadcast_url field",
		})
		return
	}

	if broadcast_url != nil {
		value := strings.TrimSpace(*broadcast_url)
		if value != "https://d3529bfd1f67.eu-west-1.playback.live-video.net/api/video/v1/eu-west-1.821643761950.channel.T8wnLDiQ8H3y.m3u8" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Select the configured league broadcast channel or use null to remove it"})
			return
		}

		broadcast_url = &value
	}

	result := s.GetDB().
		WithContext(c.Request.Context()).
		Model(&models.Fixture{}).
		Where("id = ?", fixtureID.String()).
		Updates(map[string]interface{}{
			"broadcast_url": broadcast_url,
			"revision":      gorm.Expr("revision + 1"),
		})

	if errors.Is(result.Error, gorm.ErrDuplicatedKey) || (result.Error != nil && strings.Contains(result.Error.Error(), "23505")) {
		c.JSON(http.StatusConflict, gin.H{"error": "This channel is attached to another fixture. Remove it there first."})
		return
	}
	if result.Error != nil {
		log.Printf("Update fixture broadcast_url failed: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update broadcast_url",
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
