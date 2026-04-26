package postgres

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDB(t *testing.T) *ContestRepository {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "..", "db", "schema.sql")),
		postgres.WithDatabase("wcwcpp-test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Wait for DB to be truly ready
	for range 10 {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, err, "failed to connect to test container db")

	t.Cleanup(func() {
		db.Close()
	})

	return NewContestRepository(db)
}

func TestContestRepository_CreateContest(t *testing.T) {
	repo := setupTestDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	uniqueSuffix := uuid.New().String()

	contest := &entity.Contest{
		Title:              "Test Contest " + uniqueSuffix,
		Slug:               "test-contest-" + uniqueSuffix,
		GroupUnlockDate:    now,
		GroupLockDate:      now.Add(time.Hour),
		KnockoutUnlockDate: now.Add(24 * time.Hour),
		KnockoutLockDate:   now.Add(48 * time.Hour),
	}

	err := repo.CreateContest(context.Background(), contest)
	require.NoError(t, err)

	assert.NotEmpty(t, contest.ID)
	// Try inserting the same contest to see if it fails (unique constraint)
	err = repo.CreateContest(context.Background(), contest)
	require.Error(t, err)
}

func TestContestRepository_CreateCountries(t *testing.T) {
	repo := setupTestDB(t)

	ctx := context.Background()
	uniqueSuffix1 := uuid.New().String()[:3]
	uniqueSuffix2 := uuid.New().String()[:3]

	countries := []entity.Country{
		{Code: uniqueSuffix1, FullName: "Country " + uniqueSuffix1},
		{Code: uniqueSuffix2, FullName: "Country " + uniqueSuffix2},
	}

	// 1. Initial insert should succeed
	err := repo.CreateCountries(ctx, countries)
	require.NoError(t, err, "initial insert should succeed")

	// 2. Re-inserting the exact same countries should succeed without error (gracefully handles existing)
	err = repo.CreateCountries(ctx, countries)
	require.NoError(t, err, "re-inserting the same countries should not error out")

	// 3. Inserting a mix of existing and new countries
	uniqueSuffix3 := uuid.New().String()[:3]
	mixedCountries := []entity.Country{
		{Code: uniqueSuffix1, FullName: "Country " + uniqueSuffix1}, // Existing
		{Code: uniqueSuffix3, FullName: "Country " + uniqueSuffix3}, // New
	}

	err = repo.CreateCountries(ctx, mixedCountries)
	require.NoError(t, err, "inserting a mix of existing and new countries should succeed")
}

func TestContestRepository_CreateMatches(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// 1. Setup a test contest
	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Match Contest " + uniqueSuffix,
		Slug:               "match-contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	err := repo.CreateContest(ctx, contest)
	require.NoError(t, err)

	// 2. Setup test countries
	c1Code := uuid.New().String()[:3]
	c2Code := uuid.New().String()[:3]
	err = repo.CreateCountries(ctx, []entity.Country{
		{Code: c1Code, FullName: "Team " + c1Code},
		{Code: c2Code, FullName: "Team " + c2Code},
	})
	require.NoError(t, err)

	// 3. Create matches
	// We need pointers to countries for the match
	country1 := &entity.Country{Code: c1Code}
	country2 := &entity.Country{Code: c2Code}

	roundIndex := 0
	matches := []entity.Match{
		{
			Round:    0,
			Country1: country1,
			Country2: country2,
		},
		{
			Round:      1,
			RoundIndex: &roundIndex,
			Country1:   nil,
			Country2:   nil,
		},
	}

	err = repo.CreateMatches(ctx, contest.ID, matches)
	require.NoError(t, err)

	// Test empty slice
	err = repo.CreateMatches(ctx, contest.ID, []entity.Match{})
	require.NoError(t, err, "empty slice should return without error")
}
