package router

import (
	"errors"
	"github.com/dilly3/edehson-league/models"
	"reflect"
	"testing"
	"time"
)

func TestCompleteFourTeamSchedule(t *testing.T) {
	tags := []string{"A", "B", "C", "D"}
	kickoff := time.Now().UTC()
	venue := "Community Ground"
	score := 1
	existing := buildSchedule(tags[:2])
	existing[0].Status = "finalized"
	existing[0].HomeScore = &score
	existing[0].AwayScore = &score
	existing[0].KickoffAt = &kickoff
	existing[0].Venue = &venue
	existing[0].Revision = 1
	snapshot := append([]models.Fixture(nil), existing...)
	for _, tc := range []struct {
		name string
		old  []models.Fixture
		want int
	}{
		{"fresh", nil, 12}, {"two to four", existing, 10}, {"three to four", buildSchedule(tags[:3]), 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			added, err := missingFixtures(tags, tc.old)
			if err != nil {
				t.Fatal(err)
			}
			if len(added) != tc.want {
				t.Fatalf("got %d additions, want %d", len(added), tc.want)
			}
			all := append(append([]models.Fixture{}, tc.old...), added...)
			days := map[int]int{}
			used := map[int]map[string]bool{}
			pairs := map[[2]string]bool{}
			for _, f := range all {
				if f.Matchday < 1 || f.Matchday > 6 {
					t.Fatal("invalid matchday")
				}
				days[f.Matchday]++
				if used[f.Matchday] == nil {
					used[f.Matchday] = map[string]bool{}
				}
				for _, team := range []string{f.HomeTeamTag, f.AwayTeamTag} {
					if used[f.Matchday][team] {
						t.Fatal("team plays twice")
					}
					used[f.Matchday][team] = true
				}
				key := [2]string{f.HomeTeamTag, f.AwayTeamTag}
				if pairs[key] {
					t.Fatal("duplicate pairing")
				}
				pairs[key] = true
			}
			for day := 1; day <= 6; day++ {
				if days[day] != 2 {
					t.Fatalf("matchday %d has %d games", day, days[day])
				}
			}
			again, err := missingFixtures(tags, all)
			if err != nil || len(again) != 0 {
				t.Fatal("repeat generation changed schedule")
			}
		})
	}
	if !reflect.DeepEqual(existing, snapshot) {
		t.Fatal("existing fixture changed")
	}
}

func TestScheduleConflicts(t *testing.T) {
	tags := []string{"A", "B", "C", "D"}
	for _, old := range [][]models.Fixture{
		{{HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 1}, {HomeTeamTag: "A", AwayTeamTag: "C", Matchday: 1}},
		{{HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 7}},
		{{HomeTeamTag: "A", AwayTeamTag: "Z", Matchday: 1}},
		{{HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 1}, {HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 2}},
		{{HomeTeamTag: "A", AwayTeamTag: "B", Matchday: 1}, {HomeTeamTag: "B", AwayTeamTag: "A", Matchday: 2}, {HomeTeamTag: "C", AwayTeamTag: "D", Matchday: 3}, {HomeTeamTag: "D", AwayTeamTag: "C", Matchday: 4}},
	} {
		if added, err := missingFixtures(tags, old); !errors.Is(err, errScheduleConflict) || added != nil {
			t.Fatal("conflicting schedule accepted")
		}
	}
	if _, err := missingFixtures(append(tags, "E"), nil); !errors.Is(err, errScheduleConflict) {
		t.Fatal("fifth team accepted")
	}
}
