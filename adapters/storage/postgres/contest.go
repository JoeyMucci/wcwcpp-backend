package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestRepository struct {
	*ContestSearcher
	db *sql.DB
}

var _ ports.ContestRepository = (*ContestRepository)(nil)

func NewContestRepository(db *sql.DB) *ContestRepository {
	return &ContestRepository{
		ContestSearcher: NewContestSearcher(db),
		db:              db,
	}
}

func (r *ContestRepository) ListContests(ctx context.Context) ([]entity.Contest, error) {
	stmt := postgres.SELECT(table.Contests.AllColumns).FROM(table.Contests)
	
	var dbContests []model.Contests
	if err := stmt.QueryContext(ctx, r.db, &dbContests); err != nil {
		return nil, err
	}
	
	var contests []entity.Contest
	for _, c := range dbContests {
		contests = append(contests, entity.Contest{
			ID:                 c.ID.String(),
			Title:              c.Title,
			Slug:               c.Slug,
			GroupUnlockDate:    c.GroupUnlockDate,
			GroupLockDate:      c.GroupLockDate,
			KnockoutUnlockDate: c.KnockoutUnlockDate,
			KnockoutLockDate:   c.KnockoutLockDate,
		})
	}
	
	return contests, nil
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


func (r *ContestRepository) CreateSubcontest(ctx context.Context, subcontest *entity.Subcontest) error {
	stmt := table.Subcontests.INSERT(
		table.Subcontests.ContestID,
		table.Subcontests.UserID,
		table.Subcontests.JoinCode,
		table.Subcontests.Title,
		table.Subcontests.Slug,
	).VALUES(
		subcontest.ContestID,
		subcontest.UserID,
		subcontest.JoinCode,
		subcontest.Title,
		subcontest.Slug,
	).RETURNING(table.Subcontests.AllColumns)

	var dest model.Subcontests
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return err
	}
	subcontest.ID = dest.ID.String()
	return nil
}

func (r *ContestRepository) JoinSubcontest(ctx context.Context, subcontestID string, userID string) error {
	stmt := table.SubcontestEntries.INSERT(
		table.SubcontestEntries.SubcontestID,
		table.SubcontestEntries.UserID,
	).VALUES(
		subcontestID,
		userID,
	)
	_, err := stmt.ExecContext(ctx, r.db)
	return err
}

func (r *ContestRepository) ListSubcontests(ctx context.Context, contestID string, userID string) ([]entity.Subcontest, error) {
	parsedUserID := uuid.MustParse(userID)
	parsedContestID := uuid.MustParse(contestID)

	stmt := postgres.SELECT(
		table.Subcontests.AllColumns,
		table.Subcontests.UserID.EQ(postgres.UUID(parsedUserID)).AS("subcontest_result.is_owner"),
		table.SubcontestEntries.UserID.IS_NOT_NULL().AS("subcontest_result.is_member"),
	).FROM(
		table.Subcontests.
			LEFT_JOIN(table.SubcontestEntries, 
				table.SubcontestEntries.SubcontestID.EQ(table.Subcontests.ID).
				AND(table.SubcontestEntries.UserID.EQ(postgres.UUID(parsedUserID))),
			),
	).WHERE(
		table.Subcontests.ContestID.EQ(postgres.UUID(parsedContestID)).
		AND(
			table.Subcontests.UserID.EQ(postgres.UUID(parsedUserID)).
			OR(table.SubcontestEntries.UserID.IS_NOT_NULL()),
		),
	)

	type SubcontestResult struct {
		model.Subcontests
		IsOwner  bool
		IsMember bool
	}

	var dest []SubcontestResult
	if err := stmt.QueryContext(ctx, r.db, &dest); err != nil {
		return nil, err
	}

	var result []entity.Subcontest
	for _, s := range dest {
		result = append(result, entity.Subcontest{
			ID:        s.ID.String(),
			ContestID: s.ContestID.String(),
			UserID:    s.UserID.String(),
			JoinCode:  s.JoinCode,
			Title:     s.Title,
			Slug:      s.Slug,
			IsOwner:   s.IsOwner,
			IsMember:  s.IsMember,
		})
	}
	return result, nil
}

func (r *ContestRepository) GetSubcontestByJoinCode(ctx context.Context, joinCode string) (*entity.Subcontest, error) {
	stmt := postgres.SELECT(table.Subcontests.AllColumns).
		FROM(table.Subcontests).
		WHERE(table.Subcontests.JoinCode.EQ(postgres.String(joinCode)))

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


func (r *ContestRepository) DeleteSubcontest(ctx context.Context, subcontestID string) error {
	stmt := table.Subcontests.DELETE().
		WHERE(table.Subcontests.ID.EQ(postgres.UUID(uuid.MustParse(subcontestID))))

	_, err := stmt.ExecContext(ctx, r.db)
	return err
}
