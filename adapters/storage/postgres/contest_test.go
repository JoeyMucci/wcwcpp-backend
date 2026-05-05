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

func TestContestRepository_GetContestBySlug(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	uniqueSuffix := uuid.New().String()
	slugStr := "test-slug-" + uniqueSuffix

	contest := &entity.Contest{
		Title:              "Test Contest " + uniqueSuffix,
		Slug:               slugStr,
		GroupUnlockDate:    now,
		GroupLockDate:      now.Add(time.Hour),
		KnockoutUnlockDate: now.Add(24 * time.Hour),
		KnockoutLockDate:   now.Add(48 * time.Hour),
	}

	err := repo.CreateContest(ctx, contest)
	require.NoError(t, err)

	// 1. Success case
	fetched, err := repo.GetContestBySlug(ctx, slugStr)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, contest.ID, fetched.ID)
	assert.Equal(t, contest.Title, fetched.Title)
	assert.Equal(t, contest.Slug, fetched.Slug)

	// 2. Not found case
	notFound, err := repo.GetContestBySlug(ctx, "non-existent-slug")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestContestRepository_CreateSubcontest(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Setup a contest
	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Contest " + uniqueSuffix,
		Slug:               "contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	err := repo.CreateContest(ctx, contest)
	require.NoError(t, err)

	// Setup a user
	userID := uuid.New().String()
	_, err = repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)", userID, userID+"@example.com", "user-"+uniqueSuffix)
	require.NoError(t, err)

	sub := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    userID,
		JoinCode:  "JOINCODE",
		Title:     "My Subcontest " + uniqueSuffix,
		Slug:      "my-subcontest-" + uniqueSuffix,
	}

	err = repo.CreateSubcontest(ctx, sub)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
}

func TestContestRepository_JoinSubcontest(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Setup a contest
	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Contest " + uniqueSuffix,
		Slug:               "contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	err := repo.CreateContest(ctx, contest)
	require.NoError(t, err)

	// Setup user 1
	user1ID := uuid.New().String()
	_, err = repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)", user1ID, user1ID+"@example.com", "user1-"+uniqueSuffix)
	require.NoError(t, err)

	// Setup user 2
	user2ID := uuid.New().String()
	_, err = repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)", user2ID, user2ID+"@example.com", "user2-"+uniqueSuffix)
	require.NoError(t, err)

	sub := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    user1ID,
		JoinCode:  "JOINCODE",
		Title:     "My Subcontest " + uniqueSuffix,
		Slug:      "my-subcontest-" + uniqueSuffix,
	}
	err = repo.CreateSubcontest(ctx, sub)
	require.NoError(t, err)

	// 1. Join with user 1
	err = repo.JoinSubcontest(ctx, sub.ID, user1ID)
	require.NoError(t, err)

	// 2. Join with user 2
	err = repo.JoinSubcontest(ctx, sub.ID, user2ID)
	require.NoError(t, err)

	// 3. Joining again should fail
	err = repo.JoinSubcontest(ctx, sub.ID, user1ID)
	require.Error(t, err)
}

func TestContestRepository_ListSubcontests(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Contest " + uniqueSuffix,
		Slug:               "contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.CreateContest(ctx, contest))

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	user3ID := uuid.New().String()

	_, err := repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)",
		user1ID, user1ID+"@example.com", "user1-"+uniqueSuffix,
		user2ID, user2ID+"@example.com", "user2-"+uniqueSuffix,
		user3ID, user3ID+"@example.com", "user3-"+uniqueSuffix,
	)
	require.NoError(t, err)

	sub1 := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    user1ID,
		JoinCode:  "CODE1111",
		Title:     "Sub1",
		Slug:      "sub1-" + uniqueSuffix,
	}
	require.NoError(t, repo.CreateSubcontest(ctx, sub1))

	sub2 := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    user2ID,
		JoinCode:  "CODE2222",
		Title:     "Sub2",
		Slug:      "sub2-" + uniqueSuffix,
	}
	require.NoError(t, repo.CreateSubcontest(ctx, sub2))

	sub3 := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    user3ID,
		JoinCode:  "CODE3333",
		Title:     "Sub3",
		Slug:      "sub3-" + uniqueSuffix,
	}
	require.NoError(t, repo.CreateSubcontest(ctx, sub3))

	// User1 joins Sub2
	require.NoError(t, repo.JoinSubcontest(ctx, sub2.ID, user1ID))

	// User1 should see Sub1 (owner, no join) and Sub2 (not owner, joined). Should not see Sub3.
	list, err := repo.ListSubcontests(ctx, contest.ID, user1ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	var fetchedSub1, fetchedSub2 *entity.Subcontest
	for _, s := range list {
		switch s.ID {
		case sub1.ID:
			v := s
			fetchedSub1 = &v
		case sub2.ID:
			v := s
			fetchedSub2 = &v
		}
	}
	require.NotNil(t, fetchedSub1)
	require.NotNil(t, fetchedSub2)

	assert.True(t, fetchedSub1.IsOwner)
	assert.False(t, fetchedSub1.IsMember)

	assert.False(t, fetchedSub2.IsOwner)
	assert.True(t, fetchedSub2.IsMember)

	// User3 should only see Sub3
	list3, err := repo.ListSubcontests(ctx, contest.ID, user3ID)
	require.NoError(t, err)
	require.Len(t, list3, 1)
	assert.Equal(t, sub3.ID, list3[0].ID)
	assert.True(t, list3[0].IsOwner)
}

func TestContestRepository_GetSubcontestByJoinCode(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Contest " + uniqueSuffix,
		Slug:               "contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.CreateContest(ctx, contest))

	userID := uuid.New().String()
	_, err := repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		userID, userID+"@example.com", "user-"+uniqueSuffix,
	)
	require.NoError(t, err)

	joinCode := "GETCODE1"
	sub := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    userID,
		JoinCode:  joinCode,
		Title:     "Sub " + uniqueSuffix,
		Slug:      "sub-" + uniqueSuffix,
	}
	require.NoError(t, repo.CreateSubcontest(ctx, sub))

	fetched, err := repo.GetSubcontestByJoinCode(ctx, joinCode)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, sub.ID, fetched.ID)

	fetchedNil, err := repo.GetSubcontestByJoinCode(ctx, "NONEXIST")
	require.NoError(t, err)
	assert.Nil(t, fetchedNil)
}

func TestContestRepository_DeleteSubcontestAndGetBySlug(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Contest " + uniqueSuffix,
		Slug:               "contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.CreateContest(ctx, contest))

	userID := uuid.New().String()
	_, err := repo.db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		userID, userID+"@example.com", "user-"+uniqueSuffix,
	)
	require.NoError(t, err)

	slugStr := "sub-" + uniqueSuffix
	sub := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    userID,
		JoinCode:  "DELCODE1",
		Title:     "Sub " + uniqueSuffix,
		Slug:      slugStr,
	}
	require.NoError(t, repo.CreateSubcontest(ctx, sub))

	fetched, err := repo.GetSubcontestBySlug(ctx, slugStr)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, sub.ID, fetched.ID)

	fetchedNil, err := repo.GetSubcontestBySlug(ctx, "NONEXISTSLUG")
	require.NoError(t, err)
	assert.Nil(t, fetchedNil)

	err = repo.DeleteSubcontest(ctx, sub.ID)
	require.NoError(t, err)

	fetchedAfterDelete, err := repo.GetSubcontestBySlug(ctx, slugStr)
	require.NoError(t, err)
	assert.Nil(t, fetchedAfterDelete)
}
