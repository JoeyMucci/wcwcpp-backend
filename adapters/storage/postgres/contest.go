package postgres

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestRepository struct {
	db *sql.DB
}

var _ ports.ContestRepository = (*ContestRepository)(nil)

func NewContestRepository(db *sql.DB) *ContestRepository {
	return &ContestRepository{db: db}
}

func (r *ContestRepository) CreateContest(ctx context.Context, contest *entity.Contest) error {
	stmt := table.Contests.INSERT(
		table.Contests.Title,
		table.Contests.Slug,
		table.Contests.GroupUnlockDate,
		table.Contests.GroupLockDate,
		table.Contests.KnockoutUnlockDate,
		table.Contests.KnockoutLockDate,
	).VALUES(
		contest.Title,
		contest.Slug,
		contest.GroupUnlockDate,
		contest.GroupLockDate,
		contest.KnockoutUnlockDate,
		contest.KnockoutLockDate,
	).RETURNING(table.Contests.AllColumns)

	var dest model.Contests
	err := stmt.QueryContext(ctx, r.db, &dest)
	if err != nil {
		return err
	}

	contest.ID = dest.ID.String()
	return nil
}

func (r *ContestRepository) CreateCountries(ctx context.Context, countries []entity.Country) error {
	if len(countries) == 0 {
		return nil
	}

	insertStmt := table.Countries.INSERT(
		table.Countries.Code,
		table.Countries.FullName,
	)

	for _, c := range countries {
		insertStmt = insertStmt.VALUES(c.Code, c.FullName)
	}

	stmt := insertStmt.ON_CONFLICT(table.Countries.Code).DO_NOTHING()

	_, err := stmt.ExecContext(ctx, r.db)
	return err
}

func (r *ContestRepository) CreateMatches(ctx context.Context, contestID string, matches []entity.Match) error {
	if len(matches) == 0 {
		return nil
	}

	stmt := postgres.SELECT(table.Countries.AllColumns).FROM(table.Countries)
	var dbCountries []model.Countries
	if err := stmt.QueryContext(ctx, r.db, &dbCountries); err != nil {
		return err
	}

	countryMap := make(map[string]string)
	for _, c := range dbCountries {
		countryMap[c.Code] = c.ID.String()
	}

	insertStmt := table.Matches.INSERT(
		table.Matches.ContestID,
		table.Matches.Round,
		table.Matches.RoundIndex,
		table.Matches.Country1ID,
		table.Matches.Country2ID,
	)

	for _, m := range matches {
		var c1ID, c2ID *string
		if m.Country1 != nil {
			id := countryMap[m.Country1.Code]
			c1ID = &id
		}
		if m.Country2 != nil {
			id := countryMap[m.Country2.Code]
			c2ID = &id
		}

		insertStmt = insertStmt.VALUES(
			contestID,
			m.Round,
			m.RoundIndex,
			c1ID,
			c2ID,
		)
	}

	_, err := insertStmt.ExecContext(ctx, r.db)
	return err
}
