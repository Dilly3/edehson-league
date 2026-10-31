package router

import (
	"testing"
	"time"

	"github.com/dilly3/edehson-league/models"
)

func TestShiftMatchdayKeepsTimesAndSkipsPlayed(t *testing.T) {
	at := func(s string) *time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return &v
	}
	fixture := models.Fixture{Model: models.Model{ID: "one"}, HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 1}
	others := []models.Fixture{
		{Model: models.Model{ID: "two"}, HomeTeamTag: "C", AwayTeamTag: "D", Matchday: 1, Status: "scheduled", KickoffAt: at("2026-10-18T17:30:00+01:00")},
		{Model: models.Model{ID: "three"}, Matchday: 2, Status: "scheduled", KickoffAt: at("2026-10-19T17:30:00+01:00")},
	}
	changed := shiftMatchday(fixture, at("2026-10-25T15:00:00+01:00"), others)
	if len(changed) != 1 || changed[0].ID != "two" {
		t.Fatalf("unexpected changed fixtures: %+v", changed)
	}
	if got := changed[0].KickoffAt.In(leagueTimezone).Format(time.RFC3339); got != "2026-10-25T17:30:00+01:00" {
		t.Fatalf("got %s", got)
	}
	if err := validateFixtureDate(fixture, at("2026-10-25T15:00:00+01:00"), others); err != nil {
		t.Fatalf("shifted matchday should validate: %v", err)
	}
}

func TestShiftMatchdayClearsUnplayedDates(t *testing.T) {
	kickoff := time.Date(2026, 10, 18, 16, 0, 0, 0, time.UTC)
	fixture := models.Fixture{Model: models.Model{ID: "one"}, Matchday: 1}
	others := []models.Fixture{
		{Model: models.Model{ID: "two"}, Matchday: 1, Status: "scheduled", KickoffAt: &kickoff},
		{Model: models.Model{ID: "done"}, Matchday: 1, Status: "finalized", KickoffAt: &kickoff},
	}
	changed := shiftMatchday(fixture, nil, others)
	if len(changed) != 1 || changed[0].ID != "two" || changed[0].KickoffAt != nil {
		t.Fatalf("unexpected changed fixtures: %+v", changed)
	}
}
