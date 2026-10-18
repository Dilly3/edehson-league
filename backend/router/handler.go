package router

import (
	"database/sql"

	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	sqldb *sql.DB
}

func NewHandler(db *gorm.DB, sqldb *sql.DB) *Handler {
	return &Handler{db: db, sqldb: sqldb}
}

func (h Handler) GetDB() *gorm.DB {
	return h.db
}

func (h Handler) GetSQLDB() *sql.DB {
	return h.sqldb
}
