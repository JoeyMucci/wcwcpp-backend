package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type PicksRepository struct {
	*Searcher
	db *sql.DB
}

var _ ports.PicksRepository = (*PicksRepository)(nil)

func NewPicksRepository(db *sql.DB) *PicksRepository {
	return &PicksRepository{
		Searcher: NewSearcher(db),
		db:       db,
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
		table.GroupPicks.ExtraQualifier,
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
		grouped[letter].ExtraQualifier = row.GroupPicks.ExtraQualifier
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
		table.GroupStandings.Rank.ASC(),
		table.GroupStandings.Points.DESC(),
		table.GroupStandings.Gd.DESC(),
		table.GroupStandings.Gf.DESC(),
		table.GroupStandings.Cs.DESC(),
		table.Countries.Code.ASC(),
	)

	var rows []dbGroupStandingRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return nil, err
	}

	result := make([]entity.GroupStanding, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.GroupStanding{
			Country:               entity.Country{Code: row.Countries.Code, FullName: row.Countries.FullName},
			Letter:                row.GroupStandings.Letter,
			Points:                int64(row.GroupStandings.Points),
			Wins:                  int64(row.GroupStandings.Wins),
			Draws:                 int64(row.GroupStandings.Draws),
			Losses:                int64(row.GroupStandings.Losses),
			GoalsFor:              int64(row.GroupStandings.Gf),
			GoalsAgainst:          int64(row.GroupStandings.Ga),
			GoalDifference:        int64(row.GroupStandings.Gd),
			ConductScore:          int64(row.GroupStandings.Cs),
			Rank:                  row.GroupStandings.Rank,
			IsThirdPlaceQualifier: row.GroupStandings.IsThirdPlaceQualifier,
		})
	}
	return result, nil
}

func (r *PicksRepository) CreateGroupPicks(ctx context.Context, userID string, contestID string, picks []entity.GroupPick) error {
	parsedUserID := uuid.MustParse(userID)
	parsedContestID := uuid.MustParse(contestID)

	countryMap, err := r.GetCountryCodeToIDMap(ctx)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Extract unique letters being submitted to delete existing picks for those specific groups
	var letters []postgres.Expression
	letterMap := make(map[string]bool)
	for _, p := range picks {
		if !letterMap[p.Letter] {
			letterMap[p.Letter] = true
			letters = append(letters, postgres.String(p.Letter))
		}
	}

	if len(letters) > 0 {
		delStmt := table.GroupPicks.DELETE().WHERE(
			table.GroupPicks.UserID.EQ(postgres.UUID(parsedUserID)).
				AND(table.GroupPicks.ContestID.EQ(postgres.UUID(parsedContestID))).
				AND(table.GroupPicks.Letter.IN(letters...)),
		)
		if _, err := delStmt.ExecContext(ctx, tx); err != nil {
			return err
		}
	}

	stmt := table.GroupPicks.INSERT(
		table.GroupPicks.UserID,
		table.GroupPicks.ContestID,
		table.GroupPicks.Letter,
		table.GroupPicks.Place,
		table.GroupPicks.CountryID,
		table.GroupPicks.ExtraQualifier,
	)

	hasValues := false
	for _, pick := range picks {
		for _, entry := range pick.Entries {
			countryIDStr, ok := countryMap[entry.Country.Code]
			if !ok {
				return fmt.Errorf("country %s not found", entry.Country.Code)
			}
			countryID := uuid.MustParse(countryIDStr)
			stmt = stmt.VALUES(
				parsedUserID,
				parsedContestID,
				pick.Letter,
				int32(entry.Place),
				countryID,
				pick.ExtraQualifier,
			)
			hasValues = true
		}
	}

	if !hasValues {
		return nil
	}

	_, err = stmt.ExecContext(ctx, tx)
	if err != nil {
		return err
	}

	// Insert record into contest_standings with 0 values if it does not exist
	standingsStmt := table.ContestStandings.INSERT(
		table.ContestStandings.ContestID,
		table.ContestStandings.UserID,
		table.ContestStandings.GroupScore,
		table.ContestStandings.KnockoutScore,
	).VALUES(
		parsedContestID,
		parsedUserID,
		postgres.Int32(0),
		postgres.Int32(0),
	).ON_CONFLICT(
		table.ContestStandings.ContestID,
		table.ContestStandings.UserID,
	).DO_NOTHING()

	if _, err := standingsStmt.ExecContext(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}

type dbKnockoutPickRow struct {
	model.KnockoutPicks
	model.Countries
}

func (r *PicksRepository) ListKnockoutPicks(ctx context.Context, userID string, contestID string) (entity.KnockoutPick, error) {
	parsedUserID := uuid.MustParse(userID)
	parsedContestID := uuid.MustParse(contestID)

	stmt := postgres.SELECT(
		table.KnockoutPicks.AllColumns,
		table.Countries.Code,
		table.Countries.FullName,
	).FROM(
		table.KnockoutPicks.INNER_JOIN(
			table.Countries, table.KnockoutPicks.CountryID.EQ(table.Countries.ID),
		),
	).WHERE(
		table.KnockoutPicks.UserID.EQ(postgres.UUID(parsedUserID)).
			AND(table.KnockoutPicks.ContestID.EQ(postgres.UUID(parsedContestID))),
	).ORDER_BY(
		table.KnockoutPicks.Round.ASC(),
		table.Countries.Code.ASC(),
	)

	var rows []dbKnockoutPickRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return entity.KnockoutPick{}, err
	}

	result := make([]entity.KnockoutPickEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.KnockoutPickEntry{
			Country: entity.Country{Code: row.Countries.Code, FullName: row.Countries.FullName},
			Round:   int(row.KnockoutPicks.Round),
		})
	}
	return entity.KnockoutPick{Entries: result}, nil
}

type dbKnockoutStandingRow struct {
	model.KnockoutStandings
	model.Countries
}

func (r *PicksRepository) ListKnockoutResults(ctx context.Context, contestID string) (entity.KnockoutPick, error) {
	parsedContestID := uuid.MustParse(contestID)

	stmt := postgres.SELECT(
		table.KnockoutStandings.AllColumns,
		table.Countries.Code,
		table.Countries.FullName,
	).FROM(
		table.KnockoutStandings.INNER_JOIN(
			table.Countries, table.KnockoutStandings.CountryID.EQ(table.Countries.ID),
		),
	).WHERE(
		table.KnockoutStandings.ContestID.EQ(postgres.UUID(parsedContestID)),
	).ORDER_BY(
		table.KnockoutStandings.Round.ASC(),
		table.Countries.Code.ASC(),
	)

	var rows []dbKnockoutStandingRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return entity.KnockoutPick{}, err
	}

	result := make([]entity.KnockoutPickEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.KnockoutPickEntry{
			Country: entity.Country{Code: row.Countries.Code, FullName: row.Countries.FullName},
			Round:   int(row.KnockoutStandings.Round),
		})
	}
	return entity.KnockoutPick{Entries: result}, nil
}

func (r *PicksRepository) CreateKnockoutPicks(ctx context.Context, userID string, contestID string, pick entity.KnockoutPick) error {
	parsedUserID := uuid.MustParse(userID)
	parsedContestID := uuid.MustParse(contestID)

	countryMap, err := r.GetCountryCodeToIDMap(ctx)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	delStmt := table.KnockoutPicks.DELETE().WHERE(
		table.KnockoutPicks.UserID.EQ(postgres.UUID(parsedUserID)).
			AND(table.KnockoutPicks.ContestID.EQ(postgres.UUID(parsedContestID))),
	)
	if _, err := delStmt.ExecContext(ctx, tx); err != nil {
		return err
	}

	stmt := table.KnockoutPicks.INSERT(
		table.KnockoutPicks.UserID,
		table.KnockoutPicks.ContestID,
		table.KnockoutPicks.Round,
		table.KnockoutPicks.CountryID,
	)

	hasValues := false
	for _, entry := range pick.Entries {
		countryIDStr, ok := countryMap[entry.Country.Code]
		if !ok {
			return fmt.Errorf("country %s not found", entry.Country.Code)
		}
		countryID := uuid.MustParse(countryIDStr)
		stmt = stmt.VALUES(
			parsedUserID,
			parsedContestID,
			int32(entry.Round),
			countryID,
		)
		hasValues = true
	}

	if !hasValues {
		return nil
	}

	_, err = stmt.ExecContext(ctx, tx)
	if err != nil {
		return err
	}

	// Insert record into contest_standings with 0 values if it does not exist
	standingsStmt := table.ContestStandings.INSERT(
		table.ContestStandings.ContestID,
		table.ContestStandings.UserID,
		table.ContestStandings.GroupScore,
		table.ContestStandings.KnockoutScore,
	).VALUES(
		parsedContestID,
		parsedUserID,
		postgres.Int32(0),
		postgres.Int32(0),
	).ON_CONFLICT(
		table.ContestStandings.ContestID,
		table.ContestStandings.UserID,
	).DO_NOTHING()

	if _, err := standingsStmt.ExecContext(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}
