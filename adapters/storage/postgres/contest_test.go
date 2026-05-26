package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContestRepository_CreateContest(t *testing.T) {
	repo := NewContestRepository(setupTestDB(t))

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
	repo := NewContestRepository(setupTestDB(t))

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
	repo := NewContestRepository(setupTestDB(t))
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

func TestContestRepository_CreateGroupStandings(t *testing.T) {
	repo := NewContestRepository(setupTestDB(t))
	ctx := context.Background()

	// Setup contest
	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "GS Contest " + uniqueSuffix,
		Slug:               "gs-contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, repo.CreateContest(ctx, contest))

	// Setup countries for two groups
	c1 := uuid.New().String()[:3]
	c2 := uuid.New().String()[:3]
	c3 := uuid.New().String()[:3]
	c4 := uuid.New().String()[:3]
	c5 := uuid.New().String()[:3]
	c6 := uuid.New().String()[:3]
	c7 := uuid.New().String()[:3]
	c8 := uuid.New().String()[:3]

	countries := []entity.Country{
		{Code: c1, FullName: "Country " + c1},
		{Code: c2, FullName: "Country " + c2},
		{Code: c3, FullName: "Country " + c3},
		{Code: c4, FullName: "Country " + c4},
		{Code: c5, FullName: "Country " + c5},
		{Code: c6, FullName: "Country " + c6},
		{Code: c7, FullName: "Country " + c7},
		{Code: c8, FullName: "Country " + c8},
	}
	require.NoError(t, repo.CreateCountries(ctx, countries))

	groups := []entity.Group{
		{Letter: "A", Countries: []entity.Country{
			{Code: c1}, {Code: c2}, {Code: c3}, {Code: c4},
		}},
		{Letter: "B", Countries: []entity.Country{
			{Code: c5}, {Code: c6}, {Code: c7}, {Code: c8},
		}},
	}

	// 1. Insert should succeed
	err := repo.CreateGroupStandings(ctx, contest.ID, groups)
	require.NoError(t, err)

	// 2. Verify correct number of rows (4 per group × 2 groups = 8)
	var count int
	err = repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM group_standings WHERE contest_id = $1", contest.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 8, count)

	// 3. Verify letter assignment — all group A rows should have letter 'A'
	var letterACount int
	err = repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM group_standings WHERE contest_id = $1 AND letter = 'A'", contest.ID,
	).Scan(&letterACount)
	require.NoError(t, err)
	assert.Equal(t, 4, letterACount)

	// 4. Verify all stats default to zero
	var nonZeroCount int
	err = repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM group_standings WHERE contest_id = $1 AND (points != 0 OR wins != 0 OR draws != 0 OR losses != 0 OR gf != 0 OR ga != 0 OR gd != 0 OR cs != 0)",
		contest.ID,
	).Scan(&nonZeroCount)
	require.NoError(t, err)
	assert.Equal(t, 0, nonZeroCount, "all stat columns should default to zero")

	// 5. Empty groups should be a no-op
	err = repo.CreateGroupStandings(ctx, contest.ID, []entity.Group{})
	require.NoError(t, err, "empty groups should return without error")
}

func TestContestRepository_GetContestBySlug(t *testing.T) {
	repo := NewContestRepository(setupTestDB(t))
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
	repo := NewContestRepository(setupTestDB(t))
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
	repo := NewContestRepository(setupTestDB(t))
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
	repo := NewContestRepository(setupTestDB(t))
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
	repo := NewContestRepository(setupTestDB(t))
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
	repo := NewContestRepository(setupTestDB(t))
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

func TestContestRepository_MatchOperations(t *testing.T) {
	repo := NewContestRepository(setupTestDB(t))
	ctx := context.Background()

	// 1. Setup a unique contest
	uniqueSuffix := uuid.New().String()
	contest := &entity.Contest{
		Title:              "Matches Contest " + uniqueSuffix,
		Slug:               "matches-contest-" + uniqueSuffix,
		GroupUnlockDate:    time.Now(),
		GroupLockDate:      time.Now().Add(time.Hour),
		KnockoutUnlockDate: time.Now().Add(24 * time.Hour),
		KnockoutLockDate:   time.Now().Add(48 * time.Hour),
	}
	err := repo.CreateContest(ctx, contest)
	require.NoError(t, err)

	// 2. Setup countries (USA, MEX, CAN, ARG)
	cCodes := []string{"USA", "MEX", "CAN", "ARG"}
	var countries []entity.Country
	for _, code := range cCodes {
		countries = append(countries, entity.Country{Code: code, FullName: "Team " + code})
	}
	err = repo.CreateCountries(ctx, countries)
	require.NoError(t, err)

	// 3. Seed group standings for Group A
	groups := []entity.Group{
		{
			Letter:    "A",
			Countries: countries,
		},
	}
	err = repo.CreateGroupStandings(ctx, contest.ID, groups)
	require.NoError(t, err)

	// 4. Create a group match (USA vs MEX) and bracket-progression knockout matches
	groupMatch := entity.Match{
		Round:    0,
		Country1: &countries[0], // USA
		Country2: &countries[1], // MEX
	}

	roundIndex := 0
	knockoutMatch := entity.Match{
		Round:      1,
		RoundIndex: &roundIndex,
	}

	nextRoundIndex := 0
	nextKnockoutMatch := entity.Match{
		Round:      2,
		RoundIndex: &nextRoundIndex,
	}

	sfIndex0 := 0
	sfMatch1 := entity.Match{
		Round:      4,
		RoundIndex: &sfIndex0,
		Country1:   &countries[0], // USA
		Country2:   &countries[1], // MEX
	}

	sfIndex1 := 1
	sfMatch2 := entity.Match{
		Round:      4,
		RoundIndex: &sfIndex1,
		Country1:   &countries[2], // CAN
		Country2:   &countries[3], // ARG
	}

	fIndex0 := 0
	finalMatch := entity.Match{
		Round:      5,
		RoundIndex: &fIndex0,
	}

	tpIndex1 := 1
	thirdPlaceMatch := entity.Match{
		Round:      5,
		RoundIndex: &tpIndex1,
	}

	err = repo.CreateMatches(ctx, contest.ID, []entity.Match{
		groupMatch,
		knockoutMatch,
		nextKnockoutMatch,
		sfMatch1,
		sfMatch2,
		finalMatch,
		thirdPlaceMatch,
	})
	require.NoError(t, err)

	// 5. List and verify Group Matches
	groupMatches, err := repo.ListGroupMatches(ctx, contest.ID, "A")
	require.NoError(t, err)
	require.Len(t, groupMatches, 1)
	assert.Equal(t, "USA", groupMatches[0].Country1.Code)
	assert.Equal(t, "MEX", groupMatches[0].Country2.Code)
	assert.Nil(t, groupMatches[0].Country1Goals)
	assert.Nil(t, groupMatches[0].Country2Goals)

	// 6. Update Group Stage Match: USA wins 3 - 0 MEX (as per scoring.md wins/points logic)
	goals1, goals2 := 3, 0
	groupMatch.Country1Goals = &goals1
	groupMatch.Country2Goals = &goals2
	err = repo.UpdateMatch(ctx, contest.ID, groupMatch)
	require.NoError(t, err)

	// Re-fetch and verify Group Match goals
	groupMatches, err = repo.ListGroupMatches(ctx, contest.ID, "A")
	require.NoError(t, err)
	require.Len(t, groupMatches, 1)
	assert.Equal(t, 3, *groupMatches[0].Country1Goals)
	assert.Equal(t, 0, *groupMatches[0].Country2Goals)

	// Retrieve the actual country UUIDs from the database
	var dbUSA, dbMEX model.Countries
	err = postgres.SELECT(table.Countries.AllColumns).FROM(table.Countries).WHERE(
		table.Countries.Code.EQ(postgres.String("USA")),
	).QueryContext(ctx, repo.db, &dbUSA)
	require.NoError(t, err)

	err = postgres.SELECT(table.Countries.AllColumns).FROM(table.Countries).WHERE(
		table.Countries.Code.EQ(postgres.String("MEX")),
	).QueryContext(ctx, repo.db, &dbMEX)
	require.NoError(t, err)

	// Verify group standings: USA has 3 points and +3 goal diff; MEX has 0 points and -3 goal diff
	var stdUSA, stdMEX model.GroupStandings
	err = postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
		table.GroupStandings.ContestID.EQ(postgres.UUID(uuid.MustParse(contest.ID))).
			AND(table.GroupStandings.CountryID.EQ(postgres.UUID(dbUSA.ID))), // USA
	).QueryContext(ctx, repo.db, &stdUSA)
	require.NoError(t, err)
	assert.Equal(t, int32(3), stdUSA.Points)
	assert.Equal(t, int32(1), stdUSA.Wins)
	assert.Equal(t, int32(3), stdUSA.Gf)
	assert.Equal(t, int32(0), stdUSA.Ga)
	assert.Equal(t, int32(3), stdUSA.Gd)

	err = postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
		table.GroupStandings.ContestID.EQ(postgres.UUID(uuid.MustParse(contest.ID))).
			AND(table.GroupStandings.CountryID.EQ(postgres.UUID(dbMEX.ID))), // MEX
	).QueryContext(ctx, repo.db, &stdMEX)
	require.NoError(t, err)
	assert.Equal(t, int32(0), stdMEX.Points)
	assert.Equal(t, int32(1), stdMEX.Losses)
	assert.Equal(t, int32(0), stdMEX.Gf)
	assert.Equal(t, int32(3), stdMEX.Ga)
	assert.Equal(t, int32(-3), stdMEX.Gd)

	// Correct the score: update to USA wins 2 - 1 MEX
	newGoals1, newGoals2 := 2, 1
	groupMatch.Country1Goals = &newGoals1
	groupMatch.Country2Goals = &newGoals2
	err = repo.UpdateMatch(ctx, contest.ID, groupMatch)
	require.NoError(t, err)

	// Verify corrected group standings
	err = postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
		table.GroupStandings.ContestID.EQ(postgres.UUID(uuid.MustParse(contest.ID))).
			AND(table.GroupStandings.CountryID.EQ(postgres.UUID(dbUSA.ID))), // USA
	).QueryContext(ctx, repo.db, &stdUSA)
	require.NoError(t, err)
	assert.Equal(t, int32(3), stdUSA.Points)
	assert.Equal(t, int32(1), stdUSA.Wins)
	assert.Equal(t, int32(2), stdUSA.Gf)
	assert.Equal(t, int32(1), stdUSA.Ga)
	assert.Equal(t, int32(1), stdUSA.Gd)

	err = postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
		table.GroupStandings.ContestID.EQ(postgres.UUID(uuid.MustParse(contest.ID))).
			AND(table.GroupStandings.CountryID.EQ(postgres.UUID(dbMEX.ID))), // MEX
	).QueryContext(ctx, repo.db, &stdMEX)
	require.NoError(t, err)
	assert.Equal(t, int32(0), stdMEX.Points)
	assert.Equal(t, int32(1), stdMEX.Losses)
	assert.Equal(t, int32(1), stdMEX.Gf)
	assert.Equal(t, int32(2), stdMEX.Ga)
	assert.Equal(t, int32(-1), stdMEX.Gd)

	// 7. List and verify Knockout Matches
	koMatches, err := repo.ListKnockoutMatches(ctx, contest.ID)
	require.NoError(t, err)
	require.Len(t, koMatches, 6)

	// Verify that the Round 1 Match is unseeded initially
	var matchR1 entity.Match
	for _, m := range koMatches {
		if m.Round == 1 && *m.RoundIndex == 0 {
			matchR1 = m
		}
	}
	assert.Nil(t, matchR1.Country1)
	assert.Nil(t, matchR1.Country2)

	// 8. Update Knockout Stage Match: CAN vs ARG, ends 2 - 2, CAN wins on penalties 4 - 3
	koGoals1, koGoals2 := 2, 2
	koPenalties1, koPenalties2 := 4, 3
	updatedKoMatch := entity.Match{
		Round:             1,
		RoundIndex:        &roundIndex,
		Country1:          &countries[2], // CAN
		Country2:          &countries[3], // ARG
		Country1Goals:     &koGoals1,
		Country2Goals:     &koGoals2,
		Country1Penalties: &koPenalties1,
		Country2Penalties: &koPenalties2,
	}

	err = repo.UpdateMatch(ctx, contest.ID, updatedKoMatch)
	require.NoError(t, err)

	// Re-fetch and verify Knockout Match details and deterministic winner progression
	koMatches, err = repo.ListKnockoutMatches(ctx, contest.ID)
	require.NoError(t, err)
	require.Len(t, koMatches, 6)

	for _, m := range koMatches {
		if m.Round == 1 && *m.RoundIndex == 0 {
			matchR1 = m
		}
	}
	assert.Equal(t, "CAN", matchR1.Country1.Code)
	assert.Equal(t, "ARG", matchR1.Country2.Code)
	assert.Equal(t, 2, *matchR1.Country1Goals)
	assert.Equal(t, 2, *matchR1.Country2Goals)
	assert.Equal(t, 4, *matchR1.Country1Penalties)
	assert.Equal(t, 3, *matchR1.Country2Penalties)

	// Verify that the winner (CAN) deterministically progressed to Round 2 Index 0 as Country1
	var matchR2 entity.Match
	for _, m := range koMatches {
		if m.Round == 2 && *m.RoundIndex == 0 {
			matchR2 = m
		}
	}
	require.NotNil(t, matchR2.Country1)
	assert.Equal(t, "CAN", matchR2.Country1.Code)
	assert.Nil(t, matchR2.Country2)

	// 9. Update Semifinal 1: USA (3) vs MEX (1) -> USA wins.
	sf1Goals1, sf1Goals2 := 3, 1
	updatedSf1 := entity.Match{
		Round:         4,
		RoundIndex:    &sfIndex0,
		Country1:      &countries[0], // USA
		Country2:      &countries[1], // MEX
		Country1Goals: &sf1Goals1,
		Country2Goals: &sf1Goals2,
	}
	err = repo.UpdateMatch(ctx, contest.ID, updatedSf1)
	require.NoError(t, err)

	// 10. Update Semifinal 2: CAN (0) vs ARG (2) -> ARG wins.
	sf2Goals1, sf2Goals2 := 0, 2
	updatedSf2 := entity.Match{
		Round:         4,
		RoundIndex:    &sfIndex1,
		Country1:      &countries[2], // CAN
		Country2:      &countries[3], // ARG
		Country1Goals: &sf2Goals1,
		Country2Goals: &sf2Goals2,
	}
	err = repo.UpdateMatch(ctx, contest.ID, updatedSf2)
	require.NoError(t, err)

	// 11. Re-fetch and verify Final and Third-Place progression
	koMatches, err = repo.ListKnockoutMatches(ctx, contest.ID)
	require.NoError(t, err)

	var finalM, thirdPlaceM entity.Match
	for _, m := range koMatches {
		if m.Round == 5 && *m.RoundIndex == 0 {
			finalM = m
		}
		if m.Round == 5 && *m.RoundIndex == 1 {
			thirdPlaceM = m
		}
	}

	// Final should be USA (winner of SF 1) vs ARG (winner of SF 2)
	require.NotNil(t, finalM.Country1)
	assert.Equal(t, "USA", finalM.Country1.Code)
	require.NotNil(t, finalM.Country2)
	assert.Equal(t, "ARG", finalM.Country2.Code)

	// Third-Place should be MEX (loser of SF 1) vs CAN (loser of SF 2)
	require.NotNil(t, thirdPlaceM.Country1)
	assert.Equal(t, "MEX", thirdPlaceM.Country1.Code)
	require.NotNil(t, thirdPlaceM.Country2)
	assert.Equal(t, "CAN", thirdPlaceM.Country2.Code)
}
