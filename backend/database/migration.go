package database

import (
	"fmt"

	"github.com/dilly3/edehson-league/membership"
	"github.com/dilly3/edehson-league/models"
	"github.com/dilly3/edehson-league/vieweraccess"
	"gorm.io/gorm"
)

func automigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.Team{}); err != nil {
		return fmt.Errorf("Could not migrate teams table: %v", err)
	}
	if err := db.AutoMigrate(&models.Player{}); err != nil {
		return fmt.Errorf("Could not migrate players table: %v", err)
	}
	if err := db.Exec(`
	CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_captain
	ON players (team_id)
	WHERE is_captain = true AND deleted_at = 0
`).Error; err != nil {
		return fmt.Errorf("could not create captain index: %w", err)
	}
	if err := db.AutoMigrate(&models.Fixture{}); err != nil {
		return fmt.Errorf("Could not migrate fixtures table: %v", err)
	}
	// AutoMigrate does not replace an existing named CHECK constraint.
	if err := db.Exec(`
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'fixtures'::regclass
          AND conname = 'chk_fixture_status'
          AND pg_get_constraintdef(oid) LIKE '%half_time%'
    ) THEN
        ALTER TABLE fixtures DROP CONSTRAINT IF EXISTS chk_fixture_status;
        ALTER TABLE fixtures ADD CONSTRAINT chk_fixture_status
            CHECK (status IN ('scheduled','live','half_time','finalized','postponed','abandoned'));
    END IF;
END $$;
`).Error; err != nil {
		return fmt.Errorf("could not migrate fixture statuses: %w", err)
	}
	if err := db.AutoMigrate(&models.MatchEvent{}); err != nil {
		return fmt.Errorf("Could not migrate match events table: %v", err)
	}
	// Existing installations previously required a scorer for every goal.
	if err := db.Exec("ALTER TABLE match_events ALTER COLUMN player_id DROP NOT NULL").Error; err != nil {
		return fmt.Errorf("could not allow unnamed goals: %w", err)
	}
	if err := db.Exec(`DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='match_events'::regclass AND conname='chk_card_requires_player') THEN
 ALTER TABLE match_events ADD CONSTRAINT chk_card_requires_player CHECK (type IN ('goal','own_goal') OR player_id IS NOT NULL);
 END IF; END $$;`).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.ResultAudit{}); err != nil {
		return fmt.Errorf("Could not migrate result audit table: %v", err)
	}
	if err := membership.Migrate(db); err != nil {
		return fmt.Errorf("could not migrate Great Minds Sport register: %w", err)
	}
	if err := vieweraccess.Migrate(db); err != nil {
		return fmt.Errorf("could not migrate viewer access: %w", err)
	}
	return nil
}
