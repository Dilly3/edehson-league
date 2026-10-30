package router

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/dilly3/edehson-league/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errPlayerHasEvents = errors.New("player has match events")
	errPlayerIsCaptain = errors.New("player is the team captain")
)

// Lock the team first, then the player. Captain changes and removals
// for the same team follow this same lock order.
func (s *Server) changeRosterPlayer(
	ctx context.Context,
	teamID string,
	playerID string,
	change func(*gorm.DB, *models.Player) error,
) error {
	return s.GetDB().WithContext(ctx).Transaction(
		func(tx *gorm.DB) error {
			var team models.Team

			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&team, "id = ?", teamID).Error; err != nil {
				return err
			}

			var player models.Player

			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND team_id = ?", playerID, teamID).
				First(&player).Error; err != nil {
				return err
			}

			return change(tx, &player)
		},
	)
}

func (s *Server) SetCaptain(c *gin.Context) {
	playerID, err := uuid.Parse(c.Param("playerID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	teamID := c.GetString("teamID")

	err = s.changeRosterPlayer(
		c.Request.Context(),
		teamID,
		playerID.String(),
		func(tx *gorm.DB, player *models.Player) error {
			// Clear the previous captain before assigning the new one.
			if err := tx.Model(&models.Player{}).
				Where(
					"team_id = ? AND is_captain = ? AND id <> ?",
					teamID,
					true,
					player.ID,
				).
				Update("is_captain", false).Error; err != nil {
				return err
			}

			return tx.Model(player).
				Update("is_captain", true).Error
		},
	)

	if playerManagementError(c, err) {
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) DeletePlayer(c *gin.Context) {
	playerID, err := uuid.Parse(c.Param("playerID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	err = s.changeRosterPlayer(
		c.Request.Context(),
		c.GetString("teamID"),
		playerID.String(),
		func(tx *gorm.DB, player *models.Player) error {
			if player.IsCaptain {
				return errPlayerIsCaptain
			}

			var eventCount int64

			if err := tx.Unscoped().
				Model(&models.MatchEvent{}).
				Where("player_id = ?", player.ID).
				Count(&eventCount).Error; err != nil {
				return err
			}

			if eventCount > 0 {
				return errPlayerHasEvents
			}

			// Uses the existing soft_delete field on models.Model.
			return tx.Delete(player).Error
		},
	)

	if playerManagementError(c, err) {
		return
	}

	c.Status(http.StatusNoContent)
}

func playerManagementError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false

	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Team or player not found",
		})

	case errors.Is(err, errPlayerIsCaptain):
		c.JSON(http.StatusConflict, gin.H{
			"error": "Assign another captain before removing this player",
		})

	case errors.Is(err, errPlayerHasEvents):
		c.JSON(http.StatusConflict, gin.H{
			"error": "This player has match records and cannot be removed",
		})

	default:
		log.Printf("Player management failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not update the team roster",
		})
	}

	return true
}
