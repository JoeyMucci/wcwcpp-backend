package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"

	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
)

type LeaderboardRepository struct {
	*Searcher
	db *sql.DB
}

var _ ports.LeaderboardRepository = (*LeaderboardRepository)(nil)

func NewLeaderboardRepository(db *sql.DB) *LeaderboardRepository {
	return &LeaderboardRepository{
		Searcher: NewSearcher(db),
		db:       db,
	}
}

func (r *LeaderboardRepository) Leaderboard(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	parsedContestID := uuid.MustParse(contestID)
	leaderboard := make(map[string][]entity.LeaderboardEntry)

	// 1. Group Standings
	stmt := postgres.SELECT(
		table.Users.Username,
		table.ContestStandings.GroupScore,
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.UUID(parsedContestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var groupDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &groupDest); err != nil {
		return nil, err
	}
	leaderboard["group"] = mapGroupDBRowsToEntity(groupDest)

	// 2. Knockout Standings
	stmt = postgres.SELECT(
		table.Users.Username,
		table.ContestStandings.KnockoutScore,
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.UUID(parsedContestID)),
	).ORDER_BY(
		table.ContestStandings.KnockoutScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var knockoutDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &knockoutDest); err != nil {
		return nil, err
	}
	leaderboard["knockout"] = mapKnockoutDBRowsToEntity(knockoutDest)

	// 3. Overall Standings
	stmt = postgres.SELECT(
		table.Users.Username,
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).AS("contest_standings.group_score"),
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.UUID(parsedContestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var overallDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &overallDest); err != nil {
		return nil, err
	}
	leaderboard["overall"] = mapGroupDBRowsToEntity(overallDest)

	return leaderboard, nil
}

func (r *LeaderboardRepository) Subleaderboard(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	parsedSubcontestID := uuid.MustParse(subcontestID)
	leaderboard := make(map[string][]entity.LeaderboardEntry)

	// All three queries join subcontest_entries → subcontests → users → contest_standings,
	// filtering contest_standings by contest_id from subcontests to avoid cross-contest contamination.
	fromClause := table.SubcontestEntries.
		INNER_JOIN(table.Subcontests, table.SubcontestEntries.SubcontestID.EQ(table.Subcontests.ID)).
		INNER_JOIN(table.Users, table.SubcontestEntries.UserID.EQ(table.Users.ID)).
		LEFT_JOIN(table.ContestStandings,
			table.SubcontestEntries.UserID.EQ(table.ContestStandings.UserID).
				AND(table.Subcontests.ContestID.EQ(table.ContestStandings.ContestID)),
		)

	// 1. Group Standings
	stmt := postgres.SELECT(
		table.Users.Username,
		postgres.COALESCE(table.ContestStandings.GroupScore, postgres.Int32(0)).AS("contest_standings.group_score"),
	).FROM(fromClause).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.UUID(parsedSubcontestID)),
	).ORDER_BY(
		postgres.COALESCE(table.ContestStandings.GroupScore, postgres.Int32(0)).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var groupDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &groupDest); err != nil {
		return nil, err
	}
	leaderboard["group"] = mapGroupDBRowsToEntity(groupDest)

	// 2. Knockout Standings
	stmt = postgres.SELECT(
		table.Users.Username,
		postgres.COALESCE(table.ContestStandings.KnockoutScore, postgres.Int32(0)).AS("contest_standings.knockout_score"),
	).FROM(fromClause).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.UUID(parsedSubcontestID)),
	).ORDER_BY(
		postgres.COALESCE(table.ContestStandings.KnockoutScore, postgres.Int32(0)).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var knockoutDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &knockoutDest); err != nil {
		return nil, err
	}
	leaderboard["knockout"] = mapKnockoutDBRowsToEntity(knockoutDest)

	// 3. Overall Standings
	stmt = postgres.SELECT(
		table.Users.Username,
		postgres.COALESCE(table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore), postgres.Int32(0)).AS("contest_standings.group_score"),
	).FROM(fromClause).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.UUID(parsedSubcontestID)),
	).ORDER_BY(
		postgres.COALESCE(table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore), postgres.Int32(0)).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var overallDest []dbLeaderboardRow
	if err := stmt.QueryContext(ctx, r.db, &overallDest); err != nil {
		return nil, err
	}
	leaderboard["overall"] = mapGroupDBRowsToEntity(overallDest)

	return leaderboard, nil
}

func (r *LeaderboardRepository) HasSubcontestAccess(ctx context.Context, userID string, subcontestSlug string) (bool, error) {
	parsedUserID := uuid.MustParse(userID)
	stmt := postgres.SELECT(table.Subcontests.AllColumns).
		FROM(table.Subcontests).
		WHERE(table.Subcontests.Slug.EQ(postgres.String(subcontestSlug)))
	var dest model.Subcontests
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return false, errors.New("subcontest not found")
		}
		return false, err
	}

	// owner
	if dest.UserID.String() == userID {
		return true, nil
	}

	stmt = postgres.SELECT(table.SubcontestEntries.AllColumns).
		FROM(table.SubcontestEntries).
		WHERE(postgres.AND(table.SubcontestEntries.SubcontestID.EQ(postgres.UUID(dest.ID)),
			table.SubcontestEntries.UserID.EQ(postgres.UUID(parsedUserID))))
	var entriesDest model.SubcontestEntries
	if err := stmt.QueryContext(ctx, r.db, &entriesDest); err != nil {
		// no rows in entries -> not a member
		if errors.Is(err, qrm.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type dbLeaderboardRow struct {
	model.Users
	model.ContestStandings
}

func mapGroupDBRowsToEntity(rows []dbLeaderboardRow) []entity.LeaderboardEntry {
	entries := make([]entity.LeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = entity.LeaderboardEntry{
			Name:  r.Username,
			Score: int64(r.GroupScore),
		}
	}
	return entries
}

func mapKnockoutDBRowsToEntity(rows []dbLeaderboardRow) []entity.LeaderboardEntry {
	entries := make([]entity.LeaderboardEntry, len(rows))
	for i, r := range rows {
		entries[i] = entity.LeaderboardEntry{
			Name:  r.Username,
			Score: int64(r.KnockoutScore),
		}
	}
	return entries
}
