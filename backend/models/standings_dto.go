package models

type StandingDTO struct {
	Position       int    `json:"position"`
	TeamTag        string `json:"team_tag"`
	TeamName       string `json:"team_name"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
}
