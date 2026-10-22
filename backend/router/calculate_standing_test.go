package router

import (
	"reflect"
	"testing"

	"github.com/dilly3/edehson-league/models"
)

func standingsFixture(
	home, away string,
	homeScore, awayScore int,
	status string,
) models.Fixture {
	return models.Fixture{
		HomeTeamTag: home,
		AwayTeamTag: away,
		HomeScore:   &homeScore,
		AwayScore:   &awayScore,
		Status:      status,
	}
}

func TestStandingsInitialOrder(t *testing.T) {
	teams := []models.Team{
		{Name: "Team C", Tag: "EDL001"},
		{Name: "Team A", Tag: "EDL003"},
		{Name: "Team B", Tag: "EDL002"},
	}

	table, err := calculateStandings(teams, nil)
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"Team A", "Team B", "Team C"}

	if len(table) != len(wantNames) {
		t.Fatalf("expected %d teams; got %d", len(wantNames), len(table))
	}

	for i, row := range table {
		if row.TeamName != wantNames[i] {
			t.Errorf(
				"position %d: expected %s; got %s",
				i+1, wantNames[i], row.TeamName,
			)
		}

		if row.Position != i+1 {
			t.Errorf("expected position %d; got %d", i+1, row.Position)
		}

		if row.Played != 0 || row.Points != 0 {
			t.Errorf("unplayed team has match totals: %+v", row)
		}
	}
}

func TestStandingsTotals(t *testing.T) {
	teams := []models.Team{
		{Name: "Team A", Tag: "EDL001"},
		{Name: "Team B", Tag: "EDL002"},
		{Name: "Team C", Tag: "EDL003"},
	}

	fixtures := []models.Fixture{
		standingsFixture("EDL001", "EDL002", 2, 0, "finalized"),
		standingsFixture("EDL002", "EDL001", 1, 1, "finalized"),

		// These fixtures must not contribute to standings.
		standingsFixture("EDL003", "EDL001", 99, 0, "scheduled"),
		standingsFixture("EDL003", "EDL002", 99, 0, "postponed"),
		standingsFixture("EDL002", "EDL003", 99, 0, "abandoned"),
	}

	got, err := calculateStandings(teams, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	want := []models.StandingDTO{
		{
			Position:       1,
			TeamTag:        "EDL001",
			TeamName:       "Team A",
			Played:         2,
			Won:            1,
			Drawn:          1,
			GoalsFor:       3,
			GoalsAgainst:   1,
			GoalDifference: 2,
			Points:         4,
		},
		{
			Position:       2,
			TeamTag:        "EDL002",
			TeamName:       "Team B",
			Played:         2,
			Drawn:          1,
			Lost:           1,
			GoalsFor:       1,
			GoalsAgainst:   3,
			GoalDifference: -2,
			Points:         1,
		},
		{
			Position: 3,
			TeamTag:  "EDL003",
			TeamName: "Team C",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected table:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestStandingsTiebreakers(t *testing.T) {
	teams := []models.Team{
		{Name: "Team A", Tag: "EDL001"},
		{Name: "Team B", Tag: "EDL002"},
		{Name: "Team C", Tag: "EDL003"},
		{Name: "Team D", Tag: "EDL004"},
	}

	tests := []struct {
		name     string
		fixtures []models.Fixture
		wantTags []string
	}{
		{
			name: "goal difference before goals scored",
			fixtures: []models.Fixture{
				standingsFixture("EDL001", "EDL002", 3, 0, "finalized"),
				standingsFixture("EDL003", "EDL004", 4, 2, "finalized"),
			},
			wantTags: []string{"EDL001", "EDL003", "EDL004", "EDL002"},
		},
		{
			name: "goals scored before alphabetical order",
			fixtures: []models.Fixture{
				standingsFixture("EDL001", "EDL002", 1, 0, "finalized"),
				standingsFixture("EDL003", "EDL004", 2, 1, "finalized"),
			},
			wantTags: []string{"EDL003", "EDL001", "EDL004", "EDL002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := calculateStandings(teams, tt.fixtures)
			if err != nil {
				t.Fatal(err)
			}

			gotTags := make([]string, 0, len(table))

			for i, row := range table {
				gotTags = append(gotTags, row.TeamTag)

				if row.Position != i+1 {
					t.Errorf("expected position %d; got %d", i+1, row.Position)
				}
			}

			if !reflect.DeepEqual(gotTags, tt.wantTags) {
				t.Fatalf("expected order %v; got %v", tt.wantTags, gotTags)
			}
		})
	}
}

func TestStandingsCorrectionRecalculatesTotals(t *testing.T) {
	teams := []models.Team{
		{Name: "Team A", Tag: "EDL001"},
		{Name: "Team B", Tag: "EDL002"},
	}

	original := standingsFixture("EDL001", "EDL002", 1, 0, "finalized")

	before, err := calculateStandings(teams, []models.Fixture{original})
	if err != nil {
		t.Fatal(err)
	}

	if before[0].Points != 3 {
		t.Fatalf("expected winner to have 3 points; got %+v", before)
	}

	corrected := standingsFixture("EDL001", "EDL002", 0, 0, "finalized")

	after, err := calculateStandings(teams, []models.Fixture{corrected})
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range after {
		if row.Played != 1 || row.Points != 1 ||
			row.Drawn != 1 || row.Won != 0 || row.Lost != 0 ||
			row.GoalsFor != 0 || row.GoalsAgainst != 0 ||
			row.GoalDifference != 0 {
			t.Errorf("correction left incorrect totals: %+v", row)
		}
	}
}
