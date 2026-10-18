package models

type Team struct {
	Model
	Name string `gorm:"size:80;not null;uniqueIndex" json:"name"`
	Tag  string `gorm:"size:6;not null;uniqueIndex" json:"tag"`
}
