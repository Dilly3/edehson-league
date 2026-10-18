package models

type Player struct {
	Model

	TeamID string `gorm:"type:uuid;not null;uniqueIndex:idx_team_player_name" json:"team_id"`

	Name string `gorm:"size:80;not null;uniqueIndex:idx_team_player_name" json:"name"`

	JerseyNumber string `gorm:"size:3;not null" json:"jersey_number"`
	Team         Team   `gorm:"foreignKey:TeamID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

type PlayerDTO struct {
	Name         string
	JerseyNumber string
	Team         string
}

func ToPlayerDTO(player Player) PlayerDTO {
	return PlayerDTO{
		Name:         player.Name,
		JerseyNumber: player.JerseyNumber,
		Team:         player.Team.Name,
	}
}

func ToPlayersDTO(players []Player) []PlayerDTO {
	result := make([]PlayerDTO, len(players))
	for ind, player := range players {
		result[ind] = ToPlayerDTO(player)

	}
	return result
}
