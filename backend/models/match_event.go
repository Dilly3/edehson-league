package models

const (
	EventGoal       = "goal"
	EventOwnGoal    = "own_goal"
	EventYellowCard = "yellow_card"
	EventRedCard    = "red_card"
)

type MatchEvent struct {
	Model

	FixtureID string `gorm:"type:uuid;not null;index" json:"fixture_id"`

	PlayerID *string `gorm:"type:uuid;index" json:"player_id"`

	TeamTag string `gorm:"size:6;not null;index" json:"team_tag"`

	Type     string `gorm:"size:16;not null;check:chk_match_event_type,type IN ('goal','own_goal','yellow_card','red_card')" json:"type"`
	Position int    `gorm:"not null;default:0" json:"position"`

	Minute *int `gorm:"check:chk_match_event_minute,minute >= 0" json:"minute"`

	Fixture Fixture `gorm:"foreignKey:FixtureID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`

	Player Player `gorm:"foreignKey:PlayerID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`

	Team Team `gorm:"foreignKey:TeamTag;references:Tag;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

type MatchEventDTO struct {
	ID         string `json:"id"`
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	TeamTag    string `json:"team_tag"`
	Type       string `json:"type"`
	Minute     *int   `json:"minute"`
	Position   int    `json:"position"`
}

func ToMatchEventDTO(event MatchEvent) MatchEventDTO {
	playerID := ""
	if event.PlayerID != nil {
		playerID = *event.PlayerID
	}
	return MatchEventDTO{
		ID:         event.ID,
		PlayerID:   playerID,
		PlayerName: event.Player.Name,
		TeamTag:    event.TeamTag,
		Type:       event.Type,
		Minute:     event.Minute,
		Position:   event.Position,
	}
}

func ToMatchEventsDTO(events []MatchEvent) []MatchEventDTO {
	result := make([]MatchEventDTO, 0, len(events))

	for _, event := range events {
		result = append(result, ToMatchEventDTO(event))
	}

	return result
}
