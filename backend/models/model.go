package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Model struct {
	ID        string                `gorm:"primaryKey" json:"id"`
	CreatedAt int64                 `json:"created_at"`
	UpdatedAt int64                 `json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `gorm:"not null;default:0" json:"deleted_at"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID != "" {
		return nil
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	m.ID = id.String()
	return nil
}
