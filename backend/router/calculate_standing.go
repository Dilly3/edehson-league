package router

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dilly3/edehson-league/models"
)

func calculateStandings(
	teams []models.Team,
	fixtures []models.Fixture,
) ([]models.StandingDTO, error) {
	byTag := make(map[string]*models.StandingDTO, len(teams))

	for _, team := range teams {
		byTag[team.Tag] = &models.StandingDTO{
			TeamTag:  team.Tag,
			TeamName: team.Name,
		}
	}

	for _, fixture := range fixtures {
		if fixture.Status != "finalized" {
			continue
		}

		if fixture.HomeScore == nil || fixture.AwayScore == nil {
			return nil, fmt.Errorf(
				"finalized fixture %s has missing scores",
				fixture.ID,
			)
		}

		home := byTag[fixture.HomeTeamTag]
		away := byTag[fixture.AwayTeamTag]

		if home == nil || away == nil || home == away {
			return nil, fmt.Errorf(
				"fixture %s has invalid teams",
				fixture.ID,
			)
		}

		homeScore := *fixture.HomeScore
		awayScore := *fixture.AwayScore

		if homeScore < 0 || awayScore < 0 {
			return nil, fmt.Errorf(
				"fixture %s has negative scores",
				fixture.ID,
			)
		}

		home.Played++
		away.Played++

		home.GoalsFor += homeScore
		home.GoalsAgainst += awayScore
		away.GoalsFor += awayScore
		away.GoalsAgainst += homeScore

		switch {
		case homeScore > awayScore:
			home.Won++
			home.Points += 3
			away.Lost++

		case homeScore < awayScore:
			away.Won++
			away.Points += 3
			home.Lost++

		default:
			home.Drawn++
			away.Drawn++
			home.Points++
			away.Points++
		}
	}

	table := make([]models.StandingDTO, 0, len(teams))

	for _, row := range byTag {
		row.GoalDifference = row.GoalsFor - row.GoalsAgainst
		table = append(table, *row)
	}

	sort.Slice(table, func(i, j int) bool {
		a, b := table[i], table[j]

		if a.Points != b.Points {
			return a.Points > b.Points
		}

		if a.GoalDifference != b.GoalDifference {
			return a.GoalDifference > b.GoalDifference
		}

		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}

		// Display tied teams consistently without breaking the tie.

		nameA := strings.ToLower(a.TeamName)
		nameB := strings.ToLower(b.TeamName)

		if nameA != nameB {
			return nameA < nameB
		}

		return a.TeamTag < b.TeamTag
	})

	for i := range table {
		table[i].Position = i + 1
	}

	return table, nil
}
