package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderboardRepository_Leaderboard(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLeaderboardRepository(db)
	ctx := context.Background()

	// 1. Create a contest
	contestID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO contests (id, title, slug, group_unlock_date, group_lock_date, knockout_unlock_date, knockout_lock_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		contestID, "World Cup 2026", "world-cup-2026", time.Now(), time.Now(), time.Now(), time.Now(),
	)
	require.NoError(t, err)

	// 2. Create users
	u1ID := uuid.New().String()
	u2ID := uuid.New().String()
	u3ID := uuid.New().String()

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, username) VALUES
		($1, 'user1@example.com', 'Alice'),
		($2, 'user2@example.com', 'Bob'),
		($3, 'user3@example.com', 'Charlie')`,
		u1ID, u2ID, u3ID,
	)
	require.NoError(t, err)

	// 3. Create contest standings
	// Alice: Group 10, Knockout 5 -> Overall 15
	// Bob: Group 8, Knockout 12 -> Overall 20
	// Charlie: Group 15, Knockout 2 -> Overall 17
	_, err = db.ExecContext(ctx, `
		INSERT INTO contest_standings (contest_id, user_id, group_score, knockout_score) VALUES
		($1, $2, 10, 5),
		($1, $3, 8, 12),
		($1, $4, 15, 2)`,
		contestID, u1ID, u2ID, u3ID,
	)
	require.NoError(t, err)

	// Test Leaderboard query with limit and offset
	result, err := repo.Leaderboard(ctx, contestID, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify group standings: Charlie (15) > Alice (10) > Bob (8)
	group := result["group"]
	require.Len(t, group, 3)
	assert.Equal(t, "Charlie", group[0].Name)
	assert.Equal(t, int64(15), group[0].Score)
	assert.Equal(t, "Alice", group[1].Name)
	assert.Equal(t, int64(10), group[1].Score)
	assert.Equal(t, "Bob", group[2].Name)
	assert.Equal(t, int64(8), group[2].Score)

	// Verify knockout standings: Bob (12) > Alice (5) > Charlie (2)
	knockout := result["knockout"]
	require.Len(t, knockout, 3)
	assert.Equal(t, "Bob", knockout[0].Name)
	assert.Equal(t, int64(12), knockout[0].Score)
	assert.Equal(t, "Alice", knockout[1].Name)
	assert.Equal(t, int64(5), knockout[1].Score)
	assert.Equal(t, "Charlie", knockout[2].Name)
	assert.Equal(t, int64(2), knockout[2].Score)

	// Verify overall standings: Bob (20) > Charlie (17) > Alice (15)
	overall := result["overall"]
	require.Len(t, overall, 3)
	assert.Equal(t, "Bob", overall[0].Name)
	assert.Equal(t, int64(20), overall[0].Score)
	assert.Equal(t, "Charlie", overall[1].Name)
	assert.Equal(t, int64(17), overall[1].Score)
	assert.Equal(t, "Alice", overall[2].Name)
	assert.Equal(t, int64(15), overall[2].Score)

	// Test pagination
	paginated, err := repo.Leaderboard(ctx, contestID, 1, 1)
	require.NoError(t, err)
	require.Len(t, paginated["group"], 1)
	assert.Equal(t, "Alice", paginated["group"][0].Name) // offset 1 should skip Charlie (15) and return Alice (10)
}

func TestLeaderboardRepository_Subleaderboard(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLeaderboardRepository(db)
	ctx := context.Background()

	// 1. Create a contest
	contestID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO contests (id, title, slug, group_unlock_date, group_lock_date, knockout_unlock_date, knockout_lock_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		contestID, "World Cup 2026", "world-cup-2026", time.Now(), time.Now(), time.Now(), time.Now(),
	)
	require.NoError(t, err)

	// 2. Create users
	u1ID := uuid.New().String()
	u2ID := uuid.New().String()
	u3ID := uuid.New().String() // not in subcontest

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, username) VALUES
		($1, 'user1@example.com', 'Alice'),
		($2, 'user2@example.com', 'Bob'),
		($3, 'user3@example.com', 'Charlie')`,
		u1ID, u2ID, u3ID,
	)
	require.NoError(t, err)

	// 3. Create subcontest owned by Alice
	subcontestID := uuid.New().String()
	_, err = db.ExecContext(ctx, `
		INSERT INTO subcontests (id, contest_id, user_id, join_code, title, slug)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		subcontestID, contestID, u1ID, "ABCDEF", "Alice Subcontest", "alice-subcontest",
	)
	require.NoError(t, err)

	// 4. Add Alice and Bob to subcontest entries (members)
	_, err = db.ExecContext(ctx, `
		INSERT INTO subcontest_entries (subcontest_id, user_id) VALUES
		($1, $2),
		($1, $3)`,
		subcontestID, u1ID, u2ID,
	)
	require.NoError(t, err)

	// 5. Create standings
	// Alice: Group 10, Knockout 5
	// Bob: Group 8, Knockout 12
	// Charlie: Group 25, Knockout 25 (not in subcontest, should not appear in subleaderboard!)
	_, err = db.ExecContext(ctx, `
		INSERT INTO contest_standings (contest_id, user_id, group_score, knockout_score) VALUES
		($1, $2, 10, 5),
		($1, $3, 8, 12),
		($1, $4, 25, 25)`,
		contestID, u1ID, u2ID, u3ID,
	)
	require.NoError(t, err)

	// Query Subleaderboard
	result, err := repo.Subleaderboard(ctx, subcontestID, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify only Alice and Bob appear
	group := result["group"]
	require.Len(t, group, 2)
	assert.Equal(t, "Alice", group[0].Name)
	assert.Equal(t, "Bob", group[1].Name)

	knockout := result["knockout"]
	require.Len(t, knockout, 2)
	assert.Equal(t, "Bob", knockout[0].Name)
	assert.Equal(t, "Alice", knockout[1].Name)
}

func TestLeaderboardRepository_HasSubcontestAccess(t *testing.T) {
	db := setupTestDB(t)
	repo := NewLeaderboardRepository(db)
	ctx := context.Background()

	// 1. Create a contest
	contestID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO contests (id, title, slug, group_unlock_date, group_lock_date, knockout_unlock_date, knockout_lock_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		contestID, "World Cup 2026", "world-cup-2026", time.Now(), time.Now(), time.Now(), time.Now(),
	)
	require.NoError(t, err)

	// 2. Create users
	ownerID := uuid.New().String()
	memberID := uuid.New().String()
	nonMemberID := uuid.New().String()

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, username) VALUES
		($1, 'owner@example.com', 'Owner'),
		($2, 'member@example.com', 'Member'),
		($3, 'nonmember@example.com', 'NonMember')`,
		ownerID, memberID, nonMemberID,
	)
	require.NoError(t, err)

	// 3. Create subcontest
	subcontestID := uuid.New().String()
	slug := "test-subcontest"
	_, err = db.ExecContext(ctx, `
		INSERT INTO subcontests (id, contest_id, user_id, join_code, title, slug)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		subcontestID, contestID, ownerID, "JOIN123", "Test Subcontest", slug,
	)
	require.NoError(t, err)

	// 4. Add member to subcontest entries
	_, err = db.ExecContext(ctx, `
		INSERT INTO subcontest_entries (subcontest_id, user_id) VALUES ($1, $2)`,
		subcontestID, memberID,
	)
	require.NoError(t, err)

	// Test owner access
	hasAccess, err := repo.HasSubcontestAccess(ctx, ownerID, slug)
	require.NoError(t, err)
	assert.True(t, hasAccess)

	// Test member access
	hasAccess, err = repo.HasSubcontestAccess(ctx, memberID, slug)
	require.NoError(t, err)
	assert.True(t, hasAccess)

	// Test non-member access
	hasAccess, err = repo.HasSubcontestAccess(ctx, nonMemberID, slug)
	require.NoError(t, err)
	assert.False(t, hasAccess)

	// Test non-existent subcontest slug
	_, err = repo.HasSubcontestAccess(ctx, ownerID, "non-existent-slug")
	require.Error(t, err)
}
