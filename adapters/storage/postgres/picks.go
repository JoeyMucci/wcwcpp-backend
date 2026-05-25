package postgres

import (
	"context"
	"database/sql"
	"sort"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type PicksRepository struct {
	*ContestSearcher
	db *sql.DB
}

var _ ports.PicksRepository = (*PicksRepository)(nil)

func NewPicksRepository(db *sql.DB) *PicksRepository {
	return &PicksRepository{
		ContestSearcher: NewContestSearcher(db),
		db:              db,
	}
}

// dbGroupPickRow is the raw row returned by the ListGroupPicks query.
type dbGroupPickRow struct {
	model.GroupPicks
	model.Countries
}

func (r *PicksRepository) ListGroupPicks(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
	parsedUserID := uuid.MustParse(userID)
	parsedContestID := uuid.MustParse(contestID)

	stmt := postgres.SELECT(
		table.GroupPicks.Letter,
		table.GroupPicks.Place,
		table.Countries.Code,
		table.Countries.FullName,
	).FROM(
		table.GroupPicks.INNER_JOIN(
			table.Countries, table.GroupPicks.CountryID.EQ(table.Countries.ID),
		),
	).WHERE(
		table.GroupPicks.UserID.EQ(postgres.UUID(parsedUserID)).
			AND(table.GroupPicks.ContestID.EQ(postgres.UUID(parsedContestID))),
	).ORDER_BY(
		table.GroupPicks.Letter.ASC(),
		table.GroupPicks.Place.ASC(),
	)

	var rows []dbGroupPickRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return nil, err
	}

	// Group rows by letter.
	grouped := make(map[string]*entity.GroupPick)
	order := make([]string, 0)
	for _, row := range rows {
		letter := row.GroupPicks.Letter
		if _, exists := grouped[letter]; !exists {
			grouped[letter] = &entity.GroupPick{Letter: letter}
			order = append(order, letter)
		}
		grouped[letter].Entries = append(grouped[letter].Entries, entity.GroupPickEntry{
			Country: entity.Country{Code: row.Countries.Code, FullName: row.Countries.FullName},
			Place:   int(row.GroupPicks.Place),
		})
	}

	// Preserve alphabetical letter order.
	sort.Strings(order)
	result := make([]entity.GroupPick, 0, len(order))
	for _, letter := range order {
		result = append(result, *grouped[letter])
	}
	return result, nil
}

// dbGroupStandingRow is the raw row returned by the ListGroupStandings query.
type dbGroupStandingRow struct {
	model.GroupStandings
	model.Countries
}

func (r *PicksRepository) ListGroupStandings(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
	parsedContestID := uuid.MustParse(contestID)

	stmt := postgres.SELECT(
		table.GroupStandings.AllColumns,
		table.Countries.Code,
		table.Countries.FullName,
	).FROM(
		table.GroupStandings.INNER_JOIN(
			table.Countries, table.GroupStandings.CountryID.EQ(table.Countries.ID),
		),
	).WHERE(
		table.GroupStandings.ContestID.EQ(postgres.UUID(parsedContestID)),
	).ORDER_BY(
		table.GroupStandings.Letter.ASC(),
		table.GroupStandings.Points.DESC(),
	)

	var rows []dbGroupStandingRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return nil, err
	}

	result := make([]entity.GroupStanding, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.GroupStanding{
			Country:        entity.Country{Code: row.Countries.Code, FullName: row.Countries.FullName},
			Letter:         row.GroupStandings.Letter,
			Points:         int64(row.GroupStandings.Points),
			Wins:           int64(row.GroupStandings.Wins),
			Draws:          int64(row.GroupStandings.Draws),
			Losses:         int64(row.GroupStandings.Losses),
			GoalsFor:       int64(row.GroupStandings.Gf),
			GoalsAgainst:   int64(row.GroupStandings.Ga),
			GoalDifference: int64(row.GroupStandings.Gd),
			ConductScore:   int64(row.GroupStandings.Cs),
		})
	}
	return result, nil
}
