package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB(host, port, user, password, dbname string) (*gorm.DB, *sql.DB, error) {
	if port == "" {
		port = "5432"
	}

	if host == "" {
		return nil, nil, fmt.Errorf("missing database detail:host")
	}
	if user == "" {
		return nil, nil, fmt.Errorf("missing database detail:user")
	}
	if password == "" {
		return nil, nil, fmt.Errorf("missing database detail:password")
	}
	if dbname == "" {
		return nil, nil, fmt.Errorf("missing database detail:dbname")
	}
	url := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(url), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Could not connect to PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("Could not access database connection pool: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	startupContext, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	err = sqlDB.PingContext(startupContext)
	cancel()

	if err != nil {
		return nil, nil, fmt.Errorf("PostgreSQL is unavailable: %v", err)
	}

	log.Println("Connected to PostgreSQL")
	err = automigrate(db)
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}
