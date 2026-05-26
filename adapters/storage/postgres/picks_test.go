package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPicksTest creates a contest + countries + group_standings (zero-seeded) and
// returns the repo, contestID, and a map of code→countryID for use in tests.
func setupPicksTest(t *testing.T) (*PicksRepository, string, map[string]string) {
	t.Helper()
	db := setupTestDB(t)
	repo := NewPicksRepository(db)
	contestRepo := NewContestRepository(db)
	ctx := context.Background()

	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Picks Contest " + uniqueSuffix,
		Slug:               "picks-contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, contestRepo.CreateContest(ctx, contest))

	// Two groups: A (c1–c4) and B (c5–c8)
	codes := make([]string, 8)
	for i := range codes {
		codes[i] = uuid.New().String()[:3]
	}
	countries := make([]entity.Country, 8)
	for i, code := range codes {
		countries[i] = entity.Country{Code: code, FullName: "Country " + code}
	}
	require.NoError(t, contestRepo.CreateCountries(ctx, countries))

	groups := []entity.Group{
		{Letter: "A", Countries: countries[:4]},
		{Letter: "B", Countries: countries[4:]},
	}
	require.NoError(t, contestRepo.CreateGroupStandings(ctx, contest.ID, groups))

	// Build code→UUID map by querying the DB
	codeToID := make(map[string]string)
	for _, code := range codes {
		var id string
		err := db.QueryRowContext(ctx, "SELECT id FROM countries WHERE code = $1", code).Scan(&id)
		require.NoError(t, err)
		codeToID[code] = id
	}

	return repo, contest.ID, codeToID
}

func TestPicksRepository_ListGroupStandings(t *testing.T) {
	repo, contestID, codeToID := setupPicksTest(t)
	ctx := context.Background()

	standings, err := repo.ListGroupStandings(ctx, contestID)
	require.NoError(t, err)

	// 2 groups × 4 countries = 8 rows
	assert.Len(t, standings, 8)

	// All stats should be zero (freshly seeded)
	for _, s := range standings {
		assert.Equal(t, int64(0), s.Points)
		assert.Equal(t, int64(0), s.Wins)
		assert.Equal(t, int64(0), s.Draws)
		assert.Equal(t, int64(0), s.Losses)
		assert.Equal(t, int64(0), s.GoalsFor)
		assert.Equal(t, int64(0), s.GoalsAgainst)
		assert.Equal(t, int64(0), s.GoalDifference)
		assert.Equal(t, int64(0), s.ConductScore)
		assert.NotEmpty(t, s.Letter)
		assert.NotEmpty(t, s.Country.Code)
	}

	// First 4 should be letter A, next 4 letter B
	for _, s := range standings[:4] {
		assert.Equal(t, "A", s.Letter)
	}
	for _, s := range standings[4:] {
		assert.Equal(t, "B", s.Letter)
	}

	// Empty contest should return empty slice
	standings2, err := repo.ListGroupStandings(ctx, uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, standings2)

	_ = codeToID
}

func TestPicksRepository_ListGroupPicks(t *testing.T) {
	repo, contestID, codeToID := setupPicksTest(t)
	ctx := context.Background()
	db := repo.db

	// Create a user
	userID := uuid.New().String()
	_, err := db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		userID, userID+"@example.com", "picker-"+userID[:6])
	require.NoError(t, err)

	// Get codes for group A and B
	var groupACodes, groupBCodes []string
	for code, _ := range codeToID {
		// We seeded A with first 4 — check letter via group_standings
		var letter string
		err := db.QueryRowContext(ctx,
			"SELECT letter FROM group_standings WHERE contest_id = $1 AND country_id = $2",
			contestID, codeToID[code],
		).Scan(&letter)
		require.NoError(t, err)
		if letter == "A" {
			groupACodes = append(groupACodes, code)
		} else {
			groupBCodes = append(groupBCodes, code)
		}
	}
	require.Len(t, groupACodes, 4)
	require.Len(t, groupBCodes, 4)

	// Insert picks for group A (place 1–4) and group B (place 1–4)
	for i, code := range groupACodes {
		_, err = db.ExecContext(ctx,
			"INSERT INTO group_picks (user_id, contest_id, country_id, letter, place) VALUES ($1, $2, $3, $4, $5)",
			userID, contestID, codeToID[code], "A", i+1,
		)
		require.NoError(t, err)
	}
	for i, code := range groupBCodes {
		_, err = db.ExecContext(ctx,
			"INSERT INTO group_picks (user_id, contest_id, country_id, letter, place) VALUES ($1, $2, $3, $4, $5)",
			userID, contestID, codeToID[code], "B", i+1,
		)
		require.NoError(t, err)
	}

	// 1. Fetch picks
	picks, err := repo.ListGroupPicks(ctx, userID, contestID)
	require.NoError(t, err)
	require.Len(t, picks, 2, "should return one GroupPick per group letter")

	assert.Equal(t, "A", picks[0].Letter)
	assert.Len(t, picks[0].Entries, 4)
	assert.Equal(t, 1, picks[0].Entries[0].Place)

	assert.Equal(t, "B", picks[1].Letter)
	assert.Len(t, picks[1].Entries, 4)

	// 2. No picks for another user should return empty
	otherUserID := uuid.New().String()
	_, err = db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		otherUserID, otherUserID+"@example.com", "other-"+otherUserID[:6])
	require.NoError(t, err)

	emptyPicks, err := repo.ListGroupPicks(ctx, otherUserID, contestID)
	require.NoError(t, err)
	assert.Empty(t, emptyPicks)
}

func TestPicksRepository_CreateGroupPicks(t *testing.T) {
	repo, contestID, codeToID := setupPicksTest(t)
	ctx := context.Background()
	db := repo.db

	userID := uuid.New().String()
	_, err := db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		userID, userID+"@example.com", "creator-"+userID[:6])
	require.NoError(t, err)

	var codes []string
	for code := range codeToID {
		codes = append(codes, code)
	}

	newPicks := []entity.GroupPick{
		{
			Letter: "A",
			Entries: []entity.GroupPickEntry{
				{Country: entity.Country{Code: codes[0]}, Place: 1},
				{Country: entity.Country{Code: codes[1]}, Place: 2},
			},
			ExtraQualifier: true,
		},
	}

	err = repo.CreateGroupPicks(ctx, userID, contestID, newPicks)
	require.NoError(t, err)

	// Direct DB query verification
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM group_picks WHERE user_id = $1 AND contest_id = $2",
		userID, contestID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPicksRepository_KnockoutPicks(t *testing.T) {
	repo, contestID, codeToID := setupPicksTest(t)
	ctx := context.Background()
	db := repo.db

	userID := uuid.New().String()
	_, err := db.ExecContext(ctx, "INSERT INTO users (id, email, username) VALUES ($1, $2, $3)",
		userID, userID+"@example.com", "kpicker-"+userID[:6])
	require.NoError(t, err)

	var codes []string
	for code := range codeToID {
		codes = append(codes, code)
	}

	pickPayload := entity.KnockoutPick{
		Entries: []entity.KnockoutPickEntry{
			{Country: entity.Country{Code: codes[0]}, Round: 16},
			{Country: entity.Country{Code: codes[1]}, Round: 8},
		},
	}

	// 1. Create picks
	err = repo.CreateKnockoutPicks(ctx, userID, contestID, pickPayload)
	require.NoError(t, err)

	// 2. Fetch picks
	picks, err := repo.ListKnockoutPicks(ctx, userID, contestID)
	require.NoError(t, err)
	require.Len(t, picks.Entries, 2)
	assert.Equal(t, 8, picks.Entries[0].Round)
	assert.Equal(t, codes[1], picks.Entries[0].Country.Code)
	assert.Equal(t, 16, picks.Entries[1].Round)
	assert.Equal(t, codes[0], picks.Entries[1].Country.Code)
}

func TestPicksRepository_ListKnockoutResults(t *testing.T) {
	repo, contestID, codeToID := setupPicksTest(t)
	ctx := context.Background()
	db := repo.db

	var codes []string
	for code := range codeToID {
		codes = append(codes, code)
	}

	// Match 1: Outright win (Country 1 wins 2-1)
	m1ID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO matches (id, contest_id, country1_id, country2_id, country1_goals, country2_goals, round, round_index)
		VALUES ($1, $2, $3, $4, 2, 1, 16, 1)`,
		m1ID, contestID, codeToID[codes[0]], codeToID[codes[1]])
	require.NoError(t, err)

	// Match 2: Penalty win (Draw 1-1, penalties 4-3 for Country 2)
	m2ID := uuid.New().String()
	_, err = db.ExecContext(ctx, `
		INSERT INTO matches (id, contest_id, country1_id, country2_id, country1_goals, country2_goals, country1_penalties, country2_penalties, round, round_index)
		VALUES ($1, $2, $3, $4, 1, 1, 3, 4, 16, 2)`,
		m2ID, contestID, codeToID[codes[2]], codeToID[codes[3]])
	require.NoError(t, err)

	results, err := repo.ListKnockoutResults(ctx, contestID)
	require.NoError(t, err)

	require.Len(t, results.Entries, 2)

	// Match 1 winner should be codes[0]
	assert.Equal(t, codes[0], results.Entries[0].Country.Code)
	assert.Equal(t, 16, results.Entries[0].Round)

	// Match 2 winner should be codes[3]
	assert.Equal(t, codes[3], results.Entries[1].Country.Code)
}

