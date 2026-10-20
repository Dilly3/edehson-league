package router

import (
	"fmt"
	"uuid"

	"github.com/dilly3/edehson-league/models"
)

type resultInput struct {
	Status    string       `json:"status"`
	HomeScore *int         `json:"home_score"`
	AwayScore *int         `json:"away_score"`
	Revision  *int         `json:"revision"`
	Events    []eventInput `json:"events"`
}

type eventInput struct {
	PlayerID string `json:"player_id"`
	Type     string `json:"type"`
	Minute   *int   `json:"minute"`
}

// roster maps player UUIDs to their participating team tags.
func validateResult(
	fixture models.Fixture,
	input resultInput,
	roster map[string]string,
) ([]models.MatchEvent, error) {
	if input.HomeScore == nil || input.AwayScore == nil {
		return nil, fmt.Errorf("home_score and away_score are required")
	}

	if *input.HomeScore < 0 || *input.AwayScore < 0 {
		return nil, fmt.Errorf("scores cannot be negative")
	}

	if input.Revision == nil || *input.Revision < 0 {
		return nil, fmt.Errorf("a nonnegative revision is required")
	}

	if len(input.Events) > 500 {
		return nil, fmt.Errorf("a result cannot contain more than 500 events")
	}

	events := make([]models.MatchEvent, 0, len(input.Events))
	yellowCards := make(map[string]int)
	redCards := make(map[string]int)

	homeGoals := 0
	awayGoals := 0
	lastKnownMinute := -1

	for index, event := range input.Events {
		position := index + 1

		playerID, err := uuid.Parse(event.PlayerID)
		if err != nil {
			return nil, fmt.Errorf(
				"event %d: invalid player ID",
				position,
			)
		}

		id := playerID.String()
		teamTag, exists := roster[id]

		if !exists ||
			(teamTag != fixture.HomeTeamTag &&
				teamTag != fixture.AwayTeamTag) {
			return nil, fmt.Errorf(
				"event %d: player must belong to a participating team",
				position,
			)
		}

		if event.Minute != nil {
			if *event.Minute < 0 {
				return nil, fmt.Errorf(
					"event %d: minute cannot be negative",
					position,
				)
			}

			if *event.Minute < lastKnownMinute {
				return nil, fmt.Errorf(
					"event %d: events must be in match order",
					position,
				)
			}

			lastKnownMinute = *event.Minute
		}

		switch event.Type {
		case models.EventGoal, models.EventOwnGoal:
			creditedTeam := teamTag

			if event.Type == models.EventOwnGoal {
				if teamTag == fixture.HomeTeamTag {
					creditedTeam = fixture.AwayTeamTag
				} else {
					creditedTeam = fixture.HomeTeamTag
				}
			}

			if creditedTeam == fixture.HomeTeamTag {
				homeGoals++
			} else {
				awayGoals++
			}

		case models.EventYellowCard:
			yellowCards[id]++

			if yellowCards[id] > 2 {
				return nil, fmt.Errorf(
					"event %d: a player cannot receive more than two yellow cards per match",
					position,
				)
			}

		case models.EventRedCard:
			redCards[id]++

			if redCards[id] > 1 {
				return nil, fmt.Errorf(
					"event %d: a player cannot receive more than one red card per match",
					position,
				)
			}

		default:
			return nil, fmt.Errorf(
				"event %d: unsupported event type",
				position,
			)
		}

		events = append(events, models.MatchEvent{
			FixtureID: fixture.ID,
			PlayerID:  id,
			TeamTag:   teamTag,
			Type:      event.Type,
			Minute:    event.Minute,
		})
	}

	if homeGoals != *input.HomeScore || awayGoals != *input.AwayScore {
		return nil, fmt.Errorf(
			"goal events produce %d–%d, but the submitted score is %d–%d",
			homeGoals,
			awayGoals,
			*input.HomeScore,
			*input.AwayScore,
		)
	}

	return events, nil
}
