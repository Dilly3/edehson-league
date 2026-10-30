package models

import "time"

type Fixture struct {
	Model

	HomeTeamTag string `gorm:"size:6;not null;uniqueIndex:idx_fixture_pair" json:"home_team_tag"`

	AwayTeamTag string `gorm:"size:6;not null;uniqueIndex:idx_fixture_pair;check:chk_different_teams,home_team_tag <> away_team_tag" json:"away_team_tag"`

	Matchday int `gorm:"not null;check:chk_positive_matchday,matchday > 0" json:"matchday"`

	KickoffAt    *time.Time `gorm:"type:timestamptz;index" json:"kickoff_at"`
	BroadcastURL *string    `gorm:"size:512;uniqueIndex" json:"broadcast_url"`
	Venue        *string    `gorm:"size:120" json:"venue"`
	Status       string     `gorm:"size:12;not null;default:scheduled;check:chk_fixture_status,status IN ('scheduled','live','half_time','finalized','postponed','abandoned')" json:"status"`

	HomeScore *int `json:"home_score"`

	AwayScore *int `gorm:"check:chk_fixture_scores,(home_score IS NULL AND away_score IS NULL) OR (home_score >= 0 AND away_score >= 0 AND home_score IS NOT NULL AND away_score IS NOT NULL)" json:"away_score"`

	Revision int `gorm:"not null;default:0" json:"revision"`

	HomeTeam Team `gorm:"foreignKey:HomeTeamTag;references:Tag;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"home_team"`

	AwayTeam Team `gorm:"foreignKey:AwayTeamTag;references:Tag;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"away_team"`
}

type FixtureDTO struct {
	ID           string     `json:"id"`
	HomeTeamTag  string     `json:"home_team_tag"`
	HomeTeam     string     `json:"home_team"`
	AwayTeamTag  string     `json:"away_team_tag"`
	AwayTeam     string     `json:"away_team"`
	Matchday     int        `json:"matchday"`
	KickoffAt    *time.Time `json:"kickoff_at"`
	BroadcastURL *string    `json:"broadcast_url"`
	Venue        *string    `json:"venue"`
	HomeScore    *int       `json:"home_score"`
	AwayScore    *int       `json:"away_score"`
	Status       string     `json:"status"`
	Revision     int        `json:"revision"`
}

func ToFixtureDTO(fixture Fixture) FixtureDTO {
	return FixtureDTO{
		ID:           fixture.ID,
		HomeTeamTag:  fixture.HomeTeamTag,
		HomeTeam:     fixture.HomeTeam.Name,
		AwayTeamTag:  fixture.AwayTeamTag,
		AwayTeam:     fixture.AwayTeam.Name,
		Matchday:     fixture.Matchday,
		KickoffAt:    fixture.KickoffAt,
		HomeScore:    fixture.HomeScore,
		AwayScore:    fixture.AwayScore,
		Status:       fixture.Status,
		Revision:     fixture.Revision,
		Venue:        fixture.Venue,
		BroadcastURL: fixture.BroadcastURL,
	}
}

func ToFixturesDTO(fixtures []Fixture) []FixtureDTO {
	result := make([]FixtureDTO, len(fixtures))
	for ind, fixture := range fixtures {
		result[ind] = ToFixtureDTO(fixture)

	}
	return result
}
