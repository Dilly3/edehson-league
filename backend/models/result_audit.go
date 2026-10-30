package models

type ResultAudit struct {
	Model

	FixtureID string `gorm:"type:uuid;not null;index" json:"fixture_id"`

	PreviousResult string `gorm:"type:jsonb;not null" json:"-"`
	NewResult      string `gorm:"type:jsonb;not null" json:"-"`

	Fixture Fixture `gorm:"foreignKey:FixtureID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}
