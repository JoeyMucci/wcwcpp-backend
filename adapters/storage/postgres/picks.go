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
		table.GroupStandings.Points.DESC(),
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

	_, err = stmt.ExecContext(ctx, r.db)
	if err != nil {
		return err
	}

	return nil
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

type matchWinnerRow struct {
	model.Matches
	Country1Code     string
	Country1FullName string
	Country2Code     string
	Country2FullName string
}

func (r *PicksRepository) ListKnockoutResults(ctx context.Context, contestID string) (entity.KnockoutPick, error) {
	parsedContestID := uuid.MustParse(contestID)

	c1 := table.Countries.AS("country1")
	c2 := table.Countries.AS("country2")

	stmt := postgres.SELECT(
		table.Matches.AllColumns,
		c1.Code.AS("match_winner_row.country1_code"),
		c1.FullName.AS("match_winner_row.country1_full_name"),
		c2.Code.AS("match_winner_row.country2_code"),
		c2.FullName.AS("match_winner_row.country2_full_name"),
	).FROM(
		table.Matches.
			INNER_JOIN(c1, table.Matches.Country1ID.EQ(c1.ID)).
			INNER_JOIN(c2, table.Matches.Country2ID.EQ(c2.ID)),
	).WHERE(
		table.Matches.Round.GT(postgres.Int32(0)).
			AND(table.Matches.ContestID.EQ(postgres.UUID(parsedContestID))),
	).ORDER_BY(
		table.Matches.Round.ASC(),
		table.Matches.RoundIndex.ASC(),
	)

	var rows []matchWinnerRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return entity.KnockoutPick{}, err
	}

	result := make([]entity.KnockoutPickEntry, 0, len(rows))
	for _, row := range rows {
		if row.Country1Goals == nil || row.Country2Goals == nil {
			continue
		}

		g1 := *row.Country1Goals
		g2 := *row.Country2Goals

		var winnerCode string
		var winnerName string

		if g1 > g2 {
			winnerCode = row.Country1Code
			winnerName = row.Country1FullName
		} else if g2 > g1 {
			winnerCode = row.Country2Code
			winnerName = row.Country2FullName
		} else {
			// Draw: check penalties
			if row.Country1Penalties != nil && row.Country2Penalties != nil {
				p1 := *row.Country1Penalties
				p2 := *row.Country2Penalties
				if p1 > p2 {
					winnerCode = row.Country1Code
					winnerName = row.Country1FullName
				} else if p2 > p1 {
					winnerCode = row.Country2Code
					winnerName = row.Country2FullName
				}
			}
		}

		if winnerCode != "" {
			result = append(result, entity.KnockoutPickEntry{
				Country: entity.Country{
					Code:     winnerCode,
					FullName: winnerName,
				},
				Round: int(row.Round),
			})
		}
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

	_, err = stmt.ExecContext(ctx, r.db)
	if err != nil {
		return err
	}

	return nil
}
