package models

type Team struct {
	Model
	DisplayName *string `gorm:"size:12" json:"display_name"`
	Name        string  `gorm:"size:80;not null;uniqueIndex" json:"name"`
	Tag         string  `gorm:"size:6;not null;uniqueIndex" json:"tag"`
}
