package models

type PlayerStatsDTO struct {
	PlayerID               string `json:"player_id"`
	PlayerTag              string `json:"player_tag"`
	Name                   string `json:"name"`
	TeamTag                string `json:"team_tag"`
	TeamName               string `json:"team_name"`
	JerseyNumber           string `json:"jersey_number"`
	Goals                  int    `json:"goals"`
	OwnGoals               int    `json:"own_goals"`
	YellowCards            int    `json:"yellow_cards"`
	StraightRedCards       int    `json:"straight_red_cards"`
	SecondYellowDismissals int    `json:"second_yellow_dismissals"`
	Dismissals             int    `json:"dismissals"`
}
