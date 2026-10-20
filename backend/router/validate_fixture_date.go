package router

import (
	"errors"
	"fmt"
	"github.com/dilly3/edehson-league/models"
	"time"
)

var errFixtureDateConflict = errors.New("fixture date conflicts with matchday")
var leagueTimezone = time.FixedZone("WAT", 60*60)

func validateFixtureDate(fixture models.Fixture, kickoff *time.Time, others []models.Fixture) error {
	if kickoff == nil {
		return nil
	}
	date := kickoff.In(leagueTimezone).Format("2006-01-02")
	games := 1
	for _, other := range others {
		if other.ID == fixture.ID {
			continue
		}
		if other.Matchday == fixture.Matchday {
			games++
			if games > 2 {
				return fmt.Errorf("%w: a matchday can contain at most two games", errFixtureDateConflict)
			}
			if other.HomeTeamTag == fixture.HomeTeamTag || other.HomeTeamTag == fixture.AwayTeamTag || other.AwayTeamTag == fixture.HomeTeamTag || other.AwayTeamTag == fixture.AwayTeamTag {
				return fmt.Errorf("%w: a team cannot play twice in one matchday", errFixtureDateConflict)
			}
		}
		if other.KickoffAt == nil {
			continue
		}
		otherDate := other.KickoffAt.In(leagueTimezone).Format("2006-01-02")
		if otherDate == date && other.Matchday != fixture.Matchday {
			return fmt.Errorf("%w: %s is already assigned to matchday %d (WAT)", errFixtureDateConflict, date, other.Matchday)
		}
		if other.Matchday == fixture.Matchday && otherDate != date {
			return fmt.Errorf("%w: matchday %d is already scheduled for %s (WAT); clear its unplayed kickoff dates before changing the matchday date", errFixtureDateConflict, fixture.Matchday, otherDate)
		}
	}
	return nil
}
