package router

import (
	"database/sql"

	"gorm.io/gorm"
)

type Server struct {
	db    *gorm.DB
	sqldb *sql.DB
}

func NewServer(db *gorm.DB, sqldb *sql.DB) *Server {
	return &Server{db: db, sqldb: sqldb}
}

func (s *Server) GetDB() *gorm.DB {
	return s.db
}

func (s *Server) GetSQLDB() *sql.DB {
	return s.sqldb
}
