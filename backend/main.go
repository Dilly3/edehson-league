package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dilly3/edehson-league/database"
	"github.com/dilly3/edehson-league/router"
)

func main() {
	// DATABASE
	host := os.Getenv("POSTGRES_HOST")
	dbport := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	db, sqldb, err := database.ConnectDB(host, dbport, user, password, dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer sqldb.Close()

	// ROUTER
	apiport := ":" + os.Getenv("PORT")
	if apiport == "" {
		apiport = ":8011"
	}

	h := router.NewHandler(db, sqldb)
	ginrouter := router.ConnectRouter(h)

	// SERVER
	server := &http.Server{
		Addr:              apiport,
		Handler:           ginrouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("RealEdehsons League API listening on http://localhost:%s", apiport)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
