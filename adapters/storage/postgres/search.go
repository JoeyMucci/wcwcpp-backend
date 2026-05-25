package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
)

type Searcher struct {
	db *sql.DB
}

func NewSearcher(db *sql.DB) *Searcher {
	return &Searcher{db: db}
}

func (s *Searcher) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	stmt := postgres.SELECT(table.Contests.AllColumns).
		FROM(table.Contests).
		WHERE(table.Contests.Slug.EQ(postgres.String(slug)))
	var dest model.Contests
	if err := stmt.QueryContext(ctx, s.db, &dest); err != nil {
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

func (s *Searcher) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	stmt := postgres.SELECT(table.Subcontests.AllColumns).
		FROM(table.Subcontests).
		WHERE(table.Subcontests.Slug.EQ(postgres.String(slug)))

	var dest model.Subcontests
	if err := stmt.QueryContext(ctx, s.db, &dest); err != nil {
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

func (s *Searcher) GetCountryCodeToIDMap(ctx context.Context) (map[string]string, error) {
	stmt := postgres.SELECT(table.Countries.AllColumns).
		FROM(table.Countries)
	var dest []model.Countries
	if err := stmt.QueryContext(ctx, s.db, &dest); err != nil {
		return nil, err
	}
	countryMap := make(map[string]string, len(dest))
	for _, c := range dest {
		countryMap[c.Code] = c.ID.String()
	}
	return countryMap, nil
}
