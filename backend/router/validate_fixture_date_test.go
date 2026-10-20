package router

import (
	"errors"
	"github.com/dilly3/edehson-league/models"
	"testing"
	"time"
)

func TestFixtureDates(t *testing.T) {
	at := func(s string) *time.Time {
		v, e := time.Parse(time.RFC3339, s)
		if e != nil {
			t.Fatal(e)
		}
		return &v
	}
	fixture := models.Fixture{Model: models.Model{ID: "one"}, HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 1}
	other := func(day int, date *time.Time) models.Fixture {
		return models.Fixture{Model: models.Model{ID: "two"}, HomeTeamTag: "C", AwayTeamTag: "D", Matchday: day, KickoffAt: date}
	}
	for _, tc := range []struct {
		name     string
		kickoff  *time.Time
		others   []models.Fixture
		conflict bool
	}{
		{"same WAT date", at("2026-10-18T23:30:00Z"), []models.Fixture{other(1, at("2026-10-19T18:00:00+01:00"))}, false},
		{"same UTC date different WAT dates", at("2026-10-18T23:30:00Z"), []models.Fixture{other(1, at("2026-10-18T18:00:00Z"))}, true},
		{"different matchday same date", at("2026-10-18T14:00:00Z"), []models.Fixture{other(2, at("2026-10-18T18:00:00Z"))}, true},
		{"unconfirmed partner", at("2026-10-18T14:00:00Z"), []models.Fixture{other(1, nil)}, false},
		{"clear kickoff", nil, []models.Fixture{other(1, at("2026-10-18T14:00:00Z"))}, false},
		{"third game", at("2026-10-18T14:00:00Z"), []models.Fixture{other(1, nil), {Model: models.Model{ID: "three"}, Matchday: 1}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFixtureDate(fixture, tc.kickoff, tc.others)
			if errors.Is(err, errFixtureDateConflict) != tc.conflict {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
	clash := other(1, nil)
	clash.HomeTeamTag = "A"
	if err := validateFixtureDate(fixture, at("2026-10-18T14:00:00Z"), []models.Fixture{clash}); !errors.Is(err, errFixtureDateConflict) {
		t.Fatal("team double booking accepted")
	}
}
