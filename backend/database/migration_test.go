package database

import (
	"github.com/dilly3/edehson-league/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestOptionalScorerMigration(t *testing.T) {
	dsn := os.Getenv("MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = automigrate(db); err != nil {
		t.Fatal(err)
	}
	// Simulate the old required-scorer column, then verify upgrade and rerun.
	if err = db.Exec("ALTER TABLE match_events ALTER COLUMN player_id SET NOT NULL").Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = automigrate(db); err != nil {
			t.Fatal(err)
		}
	}
	var nullable string
	if err = db.Raw("SELECT is_nullable FROM information_schema.columns WHERE table_name='match_events' AND column_name='player_id'").Scan(&nullable).Error; err != nil {
		t.Fatal(err)
	}
	if nullable != "YES" {
		t.Fatal("scorer column is still required")
	}
	var count int64
	db.Raw("SELECT count(*) FROM pg_constraint WHERE conrelid='match_events'::regclass AND conname='chk_card_requires_player'").Scan(&count)
	if count != 1 {
		t.Fatal("missing card player constraint")
	}
	dto := models.ToMatchEventDTO(models.MatchEvent{Type: models.EventGoal, TeamTag: "EDL001"})
	if dto.PlayerID != "" || dto.TeamTag != "EDL001" {
		t.Fatal("unnamed goal DTO changed")
	}
}
