package models

type Player struct {
	Model

	TeamID string `gorm:"type:uuid;not null;uniqueIndex:idx_team_player_name" json:"team_id"`

	Name         string `gorm:"size:80;not null;uniqueIndex:idx_team_player_name" json:"name"`
	IsCaptain    bool   `gorm:"not null;default:false" json:"is_captain"`
	JerseyNumber string `gorm:"size:3;not null" json:"jersey_number"`
	Team         Team   `gorm:"foreignKey:TeamID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

type PlayerDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	JerseyNumber string `json:"jersey_number"`
	Team         string `json:"team"`
	TeamTag      string `json:"team_tag"`
	PlayerTag    string `json:"player_tag"`
	IsCaptain    bool   `json:"is_captain"`
}

func ToPlayerDTO(player Player) PlayerDTO {
	return PlayerDTO{
		ID:           player.ID,
		Name:         player.Name,
		JerseyNumber: player.JerseyNumber,
		Team:         player.Team.Name,
		TeamTag:      player.Team.Tag,
		PlayerTag:    player.Team.Tag + "-" + player.JerseyNumber,
		IsCaptain:    player.IsCaptain,
	}
}

func ToPlayersDTO(players []Player) []PlayerDTO {
	result := make([]PlayerDTO, len(players))
	for ind, player := range players {
		result[ind] = ToPlayerDTO(player)

	}
	return result
}
