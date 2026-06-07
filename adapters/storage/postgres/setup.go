package postgres

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	sharedDB        *sql.DB
	sharedContainer *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start a single PostgreSQL container for all tests in this package
	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "..", "db", "schema.sql")),
		postgres.WithDatabase("wcwcpp-test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	sharedContainer = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminateContainer()
		log.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		terminateContainer()
		log.Fatalf("failed to open database connection: %v", err)
	}
	sharedDB = db

	// Wait for DB to be truly ready
	for i := 0; i < 10; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		db.Close()
		terminateContainer()
		log.Fatalf("failed to connect to test container db: %v", err)
	}

	// Run all package tests
	code := m.Run()

	// Clean up resources after all tests have completed
	db.Close()
	terminateContainer()

	os.Exit(code)
}

func terminateContainer() {
	if sharedContainer != nil {
		ctx := context.Background()
		_ = sharedContainer.Terminate(ctx)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	ctx := context.Background()

	// Truncate all tables to ensure a clean state for the test
	_, err := sharedDB.ExecContext(ctx, `
		TRUNCATE TABLE 
			group_picks, 
			knockout_picks, 
			group_standings, 
			knockout_standings, 
			subcontest_entries, 
			contest_standings, 
			subcontests, 
			matches, 
			countries, 
			contests, 
			users 
		CASCADE;
	`)
	require.NoError(t, err, "failed to truncate tables for clean test state")

	return sharedDB
}
