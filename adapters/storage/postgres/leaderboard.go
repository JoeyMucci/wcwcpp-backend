package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"

	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
)

type LeaderboardRepository struct {
	db *sql.DB
}

var _ ports.LeaderboardRepository = (*LeaderboardRepository)(nil)

func NewLeaderboardRepository(db *sql.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db: db}
}

func (r *LeaderboardRepository) Leaderboard(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	leaderboard := make(map[string][]entity.LeaderboardEntry)

	stmt := postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.GroupScore.AS("Score"),
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.String(contestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))
	var dest []entity.LeaderboardEntry
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["group"] = dest
	stmt = postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.KnockoutScore.AS("Score"),
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.String(contestID)),
	).ORDER_BY(
		table.ContestStandings.KnockoutScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["knockout"] = dest
	stmt = postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).AS("Score"),
	).FROM(
		table.ContestStandings.INNER_JOIN(table.Users, table.ContestStandings.UserID.EQ(table.Users.ID)),
	).WHERE(
		table.ContestStandings.ContestID.EQ(postgres.String(contestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["overall"] = dest

	return leaderboard, nil
}

func (r *LeaderboardRepository) Subleaderboard(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	leaderboard := make(map[string][]entity.LeaderboardEntry)

	stmt := postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.GroupScore.AS("Score"),
	).FROM(
		table.SubcontestEntries.INNER_JOIN(table.Users, table.SubcontestEntries.UserID.EQ(table.Users.ID)).INNER_JOIN(table.ContestStandings, table.SubcontestEntries.UserID.EQ(table.ContestStandings.UserID)),
	).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.String(subcontestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))

	var dest []entity.LeaderboardEntry
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["group"] = dest

	stmt = postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.KnockoutScore.AS("Score"),
	).FROM(
		table.SubcontestEntries.INNER_JOIN(table.Users, table.SubcontestEntries.UserID.EQ(table.Users.ID)).INNER_JOIN(table.ContestStandings, table.SubcontestEntries.UserID.EQ(table.ContestStandings.UserID)),
	).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.String(subcontestID)),
	).ORDER_BY(
		table.ContestStandings.KnockoutScore.DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["knockout"] = dest

	stmt = postgres.SELECT(
		table.Users.Username.AS("Name"),
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).AS("Score"),
	).FROM(
		table.SubcontestEntries.INNER_JOIN(table.Users, table.SubcontestEntries.UserID.EQ(table.Users.ID)).INNER_JOIN(table.ContestStandings, table.SubcontestEntries.UserID.EQ(table.ContestStandings.UserID)),
	).WHERE(
		table.SubcontestEntries.SubcontestID.EQ(postgres.String(subcontestID)),
	).ORDER_BY(
		table.ContestStandings.GroupScore.ADD(table.ContestStandings.KnockoutScore).DESC(),
	).LIMIT(int64(limit)).OFFSET(int64(offset))
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}
	leaderboard["overall"] = dest

	return leaderboard, nil
}

func (r *LeaderboardRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	stmt := postgres.SELECT(table.Contests.AllColumns).
		FROM(table.Contests).
		WHERE(table.Contests.Slug.EQ(postgres.String(slug)))
	var dest model.Contests
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entity.Contest{
		ID:                 dest.ID.String(),
		Title:              dest.Title,
		Slug:               dest.Slug,
		GroupUnlockDate:    dest.GroupUnlockDate,
		GroupLockDate:      dest.GroupLockDate,
		KnockoutUnlockDate: dest.KnockoutUnlockDate,
		KnockoutLockDate:   dest.KnockoutLockDate,
	}, nil
}

func (r *LeaderboardRepository) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	stmt := postgres.SELECT(table.Subcontests.AllColumns).
		FROM(table.Subcontests).
		WHERE(table.Subcontests.Slug.EQ(postgres.String(slug)))

	var dest model.Subcontests
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entity.Subcontest{
		ID:        dest.ID.String(),
		ContestID: dest.ContestID.String(),
		UserID:    dest.UserID.String(),
		JoinCode:  dest.JoinCode,
		Title:     dest.Title,
		Slug:      dest.Slug,
	}, nil
}
