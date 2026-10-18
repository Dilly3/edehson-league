package models

import "time"

type Fixture struct {
	Model

	HomeTeamTag string `gorm:"size:6;not null;uniqueIndex:idx_fixture_pair" json:"home_team_tag"`

	AwayTeamTag string `gorm:"size:6;not null;uniqueIndex:idx_fixture_pair;check:chk_different_teams,home_team_tag <> away_team_tag" json:"away_team_tag"`

	Matchday int `gorm:"not null;check:chk_positive_matchday,matchday > 0" json:"matchday"`

	KickoffAt *time.Time `gorm:"type:timestamptz;index" json:"kickoff_at"`

	Status string `gorm:"size:12;not null;default:scheduled;check:chk_fixture_status,status IN ('scheduled','finalized','postponed','abandoned')" json:"status"`

	HomeScore *int `json:"home_score"`

	AwayScore *int `gorm:"check:chk_fixture_scores,(home_score IS NULL AND away_score IS NULL) OR (home_score >= 0 AND away_score >= 0 AND home_score IS NOT NULL AND away_score IS NOT NULL)" json:"away_score"`

	Revision int `gorm:"not null;default:0" json:"revision"`

	HomeTeam Team `gorm:"foreignKey:HomeTeamTag;references:Tag;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"home_team"`

	AwayTeam Team `gorm:"foreignKey:AwayTeamTag;references:Tag;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"away_team"`
}
