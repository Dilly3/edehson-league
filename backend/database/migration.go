package database

import (
	"fmt"

	"github.com/dilly3/edehson-league/models"
	"gorm.io/gorm"
)

func automigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.Team{}); err != nil {
		return fmt.Errorf("Could not migrate teams table: %v", err)
	}
	if err := db.AutoMigrate(&models.Player{}); err != nil {
		return fmt.Errorf("Could not migrate players table: %v", err)
	}
	if err := db.AutoMigrate(&models.Fixture{}); err != nil {
		return fmt.Errorf("Could not migrate players table: %v", err)
	}
	return nil
}
