package router

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type resultSnapshot struct {
	HomeScore *int                `json:"home_score"`
	AwayScore *int                `json:"away_score"`
	Status    string              `json:"status"`
	Revision  int                 `json:"revision"`
	Events    []models.MatchEvent `json:"events"`
}

func (s *Server) EditResult(c *gin.Context) {
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
		256*1024,
	)

	var input resultInput

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid result JSON or unsupported fields",
		})
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide exactly one JSON object",
		})
		return
	}

	if input.Revision == nil || *input.Revision < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "A nonnegative revision is required",
		})
		return
	}

	errStaleResult := errors.New("stale result")
	errInvalidStatus := errors.New("invalid fixture status")
	nextStatus, statusErr := requestedResultStatus(input.Status)
	if statusErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": statusErr.Error()})
		return
	}
	var validationErr error
	var newRevision int

	err = s.GetDB().WithContext(c.Request.Context()).Transaction(
		func(tx *gorm.DB) error {
			var fixture models.Fixture

			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&fixture, "id = ?", fixtureID.String()).
				Error; err != nil {
				return err
			}

			if fixture.Revision != *input.Revision {
				return errStaleResult
			}

			if !canPublishStatus(fixture.Status, nextStatus) {
				return errInvalidStatus
			}

			var teams []models.Team
			if err := tx.
				Where(
					"tag IN ?",
					[]string{
						fixture.HomeTeamTag,
						fixture.AwayTeamTag,
					},
				).
				Find(&teams).Error; err != nil {
				return err
			}

			teamTags := make(map[string]string)
			teamIDs := make([]string, 0, len(teams))

			for _, team := range teams {
				teamTags[team.ID] = team.Tag
				teamIDs = append(teamIDs, team.ID)
			}

			var players []models.Player
			if err := tx.
				Clauses(clause.Locking{Strength: "SHARE"}).
				Where("team_id IN ?", teamIDs).
				Find(&players).Error; err != nil {
				return err
			}

			roster := make(map[string]string)
			for _, player := range players {
				roster[player.ID] = teamTags[player.TeamID]
			}

			events, err := validateResult(fixture, input, roster)
			if err != nil {
				validationErr = err
				return err
			}

			for i := range events {
				events[i].Position = i + 1
			}

			previousEvents := make([]models.MatchEvent, 0)
			if err := tx.
				Where("fixture_id = ?", fixture.ID).
				Order("position ASC").
				Find(&previousEvents).Error; err != nil {
				return err
			}

			previousJSON, err := json.Marshal(resultSnapshot{
				HomeScore: fixture.HomeScore,
				AwayScore: fixture.AwayScore,
				Status:    fixture.Status,
				Revision:  fixture.Revision,
				Events:    previousEvents,
			})
			if err != nil {
				return err
			}

			// Corrections replace events. Their previous versions
			// are preserved in the audit record below.
			if err := tx.Unscoped().
				Where("fixture_id = ?", fixture.ID).
				Delete(&models.MatchEvent{}).Error; err != nil {
				return err
			}

			if len(events) > 0 {
				if err := tx.Create(&events).Error; err != nil {
					return err
				}
			}

			newRevision = fixture.Revision + 1

			if err := tx.Model(&fixture).
				Updates(map[string]interface{}{
					"home_score": *input.HomeScore,
					"away_score": *input.AwayScore,
					"status":     nextStatus,
					"revision":   newRevision,
				}).Error; err != nil {
				return err
			}

			nextJSON, err := json.Marshal(resultSnapshot{
				HomeScore: input.HomeScore,
				AwayScore: input.AwayScore,
				Status:    nextStatus,
				Revision:  newRevision,
				Events:    events,
			})
			if err != nil {
				return err
			}

			return tx.Create(&models.ResultAudit{
				FixtureID:      fixture.ID,
				PreviousResult: string(previousJSON),
				NewResult:      string(nextJSON),
			}).Error
		},
	)

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fixture not found",
		})
	case errors.Is(err, errStaleResult):
		c.JSON(http.StatusConflict, gin.H{
			"error": "Fixture changed; refresh before submitting again",
		})
	case errors.Is(err, errInvalidStatus):
		c.JSON(http.StatusConflict, gin.H{
			"error": "This fixture cannot accept the selected status. Full-time matches can only be corrected as full time",
		})
	case validationErr != nil:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": validationErr.Error(),
		})
	case err != nil:
		log.Printf("Save result failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not save result",
		})
	default:
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"fixture_id": fixtureID.String(),
				"home_score": *input.HomeScore,
				"away_score": *input.AwayScore,
				"status":     nextStatus,
				"revision":   newRevision,
			},
		})
	}
}
