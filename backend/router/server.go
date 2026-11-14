package router

import (
	"database/sql"
	"errors"

	"github.com/dilly3/edehson-league/adminauth"
	"gorm.io/gorm"
)

type Server struct {
	db         *gorm.DB
	sqldb      *sql.DB
	admintoken *string
	auth       *adminauth.Auth
}

func NewServer(db *gorm.DB, sqldb *sql.DB, adminToken string) (*Server, error) {
	if adminToken == "" {
		return nil, errors.New("empty admin token")
	}
	return &Server{db: db, sqldb: sqldb, admintoken: &adminToken}, nil
}

func (s *Server) GetDB() *gorm.DB {
	return s.db
}

func (s *Server) GetAdminToken() string {
	return *s.admintoken
}

func (s *Server) GetSQLDB() *sql.DB {
	return s.sqldb
}
