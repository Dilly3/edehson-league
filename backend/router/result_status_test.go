package router

import (
	"github.com/dilly3/edehson-league/models"
	"testing"
)

func TestResultStatuses(t *testing.T) {
	for _, s := range []string{"live", "half_time", "finalized"} {
		got, err := requestedResultStatus(s)
		if err != nil || got != s {
			t.Fatalf("status %s rejected", s)
		}
	}
	if got, err := requestedResultStatus(""); err != nil || got != "finalized" {
		t.Fatal("legacy final publishing changed")
	}
	if _, err := requestedResultStatus("scheduled"); err == nil {
		t.Fatal("invalid publication accepted")
	}
	for _, tc := range []struct {
		from, to string
		want     bool
	}{
		{"scheduled", "live", true}, {"live", "half_time", true}, {"half_time", "live", true}, {"live", "finalized", true},
		{"finalized", "live", false}, {"finalized", "half_time", false}, {"finalized", "finalized", true}, {"postponed", "live", false}, {"abandoned", "finalized", false},
	} {
		if canPublishStatus(tc.from, tc.to) != tc.want {
			t.Fatalf("incorrect transition %s -> %s", tc.from, tc.to)
		}
	}
}
func TestLiveResultsExcludedFromStandings(t *testing.T) {
	one := 1
	zero := 0
	teams := []models.Team{{Name: "A", Tag: "A"}, {Name: "B", Tag: "B"}}
	for _, status := range []string{"live", "half_time"} {
		rows, err := calculateStandings(teams, []models.Fixture{{HomeTeamTag: "A", AwayTeamTag: "B", HomeScore: &one, AwayScore: &zero, Status: status}})
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.Played != 0 || row.Points != 0 {
				t.Fatal("in-progress score counted in standings")
			}
		}
	}
}
