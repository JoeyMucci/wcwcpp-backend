package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// NewDB creates and tests a connection to a PostgreSQL database using DATABASE_URL.
func NewDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is highly required but not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool parameters for production readiness
	db.SetMaxOpenConns(25)                 // Set maximum number of open connections
	db.SetMaxIdleConns(25)                 // Set maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum amount of time a connection may be reused
	db.SetConnMaxIdleTime(3 * time.Minute) // Maximum amount of time a connection may be idle

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to the database")
	return db, nil
}
