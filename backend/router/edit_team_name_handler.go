package router

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func (s *Server) EditTeamName(c *gin.Context) {
	tag := strings.ToUpper(strings.TrimSpace(c.Param("tag")))
	if !regexp.MustCompile(`^EDL[0-9]{3}$`).MatchString(tag) || tag == "EDL000" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team tag"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var input struct {
		Name        string          `json:"name"`
		DisplayName json.RawMessage `json:"display_name"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide only a team name"})
		return
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide exactly one JSON object"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Team name must contain 2–20 characters"})
		return
	}
	updates := map[string]interface{}{"name": name}
	if len(input.DisplayName) > 0 {
		var raw *string
		if err := json.Unmarshal(input.DisplayName, &raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Display name must be a string or null"})
			return
		}
		displayName, err := normalizeTeamDisplayName(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates["display_name"] = displayName
	}
	// Update team names only: identifiers and all relationships stay stable.
	result := s.GetDB().WithContext(c.Request.Context()).Model(&models.Team{}).Where("tag = ?", tag).Updates(updates)
	var pgErr *pgconn.PgError
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) || (errors.As(result.Error, &pgErr) && pgErr.Code == "23505") {
		c.JSON(http.StatusConflict, gin.H{"error": "A team with this name already exists"})
		return
	}
	if result.Error != nil {
		log.Printf("Rename team failed: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not rename team"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
