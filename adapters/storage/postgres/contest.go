package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/model"
	"github.com/joey/wcwcpp-backend/adapters/storage/jet/wcwcpp/public/table"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestRepository struct {
	*Searcher
	db *sql.DB
}

var _ ports.ContestRepository = (*ContestRepository)(nil)

func NewContestRepository(db *sql.DB) *ContestRepository {
	return &ContestRepository{
		Searcher: NewSearcher(db),
		db:       db,
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

func (r *ContestRepository) CreateGroupStandings(ctx context.Context, contestID string, groups []entity.Group) error {
	if len(groups) == 0 {
		return nil
	}

	fmt.Println("Creating group standings for contest", contestID)
	countryMap, err := r.GetCountryCodeToIDMap(ctx)
	fmt.Println(countryMap)
	if err != nil {
		return err
	}

	insertStmt := table.GroupStandings.INSERT(
		table.GroupStandings.ContestID,
		table.GroupStandings.CountryID,
		table.GroupStandings.Letter,
	)
	for _, g := range groups {
		for _, c := range g.Countries {
			countryID, ok := countryMap[c.Code]
			if !ok {
				continue
			}
			insertStmt = insertStmt.VALUES(contestID, countryID, g.Letter)
		}
	}

	_, err = insertStmt.ExecContext(ctx, r.db)
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

type dbMatchRow struct {
	model.Matches
	Country1Code     *string
	Country1FullName *string
	Country2Code     *string
	Country2FullName *string
}

func mapDbRowsToMatches(rows []dbMatchRow) []entity.Match {
	matches := make([]entity.Match, 0, len(rows))
	for _, row := range rows {
		var c1, c2 *entity.Country
		if row.Country1Code != nil && row.Country1FullName != nil {
			c1 = &entity.Country{
				Code:     *row.Country1Code,
				FullName: *row.Country1FullName,
			}
		}
		if row.Country2Code != nil && row.Country2FullName != nil {
			c2 = &entity.Country{
				Code:     *row.Country2Code,
				FullName: *row.Country2FullName,
			}
		}

		var c1g, c2g, c1p, c2p *int
		if row.Country1Goals != nil {
			val := int(*row.Country1Goals)
			c1g = &val
		}
		if row.Country2Goals != nil {
			val := int(*row.Country2Goals)
			c2g = &val
		}
		if row.Country1Penalties != nil {
			val := int(*row.Country1Penalties)
			c1p = &val
		}
		if row.Country2Penalties != nil {
			val := int(*row.Country2Penalties)
			c2p = &val
		}

		var c1cs, c2cs *int
		if row.Country1ConductScore != nil {
			val := int(*row.Country1ConductScore)
			c1cs = &val
		}
		if row.Country2ConductScore != nil {
			val := int(*row.Country2ConductScore)
			c2cs = &val
		}

		var roundIndex *int
		if row.RoundIndex != nil {
			val := int(*row.RoundIndex)
			roundIndex = &val
		}

		matches = append(matches, entity.Match{
			Country1:             c1,
			Country2:             c2,
			Country1Goals:        c1g,
			Country2Goals:        c2g,
			Country1Penalties:    c1p,
			Country2Penalties:    c2p,
			Country1ConductScore: c1cs,
			Country2ConductScore: c2cs,
			Round:                int(row.Round),
			RoundIndex:           roundIndex,
		})
	}
	return matches
}

func (r *ContestRepository) ListGroupMatches(ctx context.Context, contestID string, letter string) ([]entity.Match, error) {
	parsedContestID := uuid.MustParse(contestID)
	c1 := table.Countries.AS("country1")
	c2 := table.Countries.AS("country2")

	stmt := postgres.SELECT(
		table.Matches.AllColumns,
		c1.Code.AS("db_match_row.country1_code"),
		c1.FullName.AS("db_match_row.country1_full_name"),
		c2.Code.AS("db_match_row.country2_code"),
		c2.FullName.AS("db_match_row.country2_full_name"),
	).FROM(
		table.Matches.
			LEFT_JOIN(c1, table.Matches.Country1ID.EQ(c1.ID)).
			LEFT_JOIN(c2, table.Matches.Country2ID.EQ(c2.ID)).
			INNER_JOIN(table.GroupStandings, table.GroupStandings.ContestID.EQ(table.Matches.ContestID).
				AND(table.GroupStandings.CountryID.EQ(table.Matches.Country1ID))),
	).WHERE(
		table.Matches.Round.EQ(postgres.Int32(0)).
			AND(table.Matches.ContestID.EQ(postgres.UUID(parsedContestID))).
			AND(table.GroupStandings.Letter.EQ(postgres.String(letter))),
	)

	var rows []dbMatchRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return nil, err
	}

	return mapDbRowsToMatches(rows), nil
}

func (r *ContestRepository) ListKnockoutMatches(ctx context.Context, contestID string) ([]entity.Match, error) {
	parsedContestID := uuid.MustParse(contestID)
	c1 := table.Countries.AS("country1")
	c2 := table.Countries.AS("country2")

	stmt := postgres.SELECT(
		table.Matches.AllColumns,
		c1.Code.AS("db_match_row.country1_code"),
		c1.FullName.AS("db_match_row.country1_full_name"),
		c2.Code.AS("db_match_row.country2_code"),
		c2.FullName.AS("db_match_row.country2_full_name"),
	).FROM(
		table.Matches.
			LEFT_JOIN(c1, table.Matches.Country1ID.EQ(c1.ID)).
			LEFT_JOIN(c2, table.Matches.Country2ID.EQ(c2.ID)),
	).WHERE(
		table.Matches.Round.GT(postgres.Int32(0)).
			AND(table.Matches.ContestID.EQ(postgres.UUID(parsedContestID))),
	).ORDER_BY(
		table.Matches.Round.ASC(),
		table.Matches.RoundIndex.ASC(),
	)

	var rows []dbMatchRow
	if err := stmt.QueryContext(ctx, r.db, &rows); err != nil {
		return nil, err
	}

	return mapDbRowsToMatches(rows), nil
}

func (r *ContestRepository) UpdateMatch(ctx context.Context, contestID string, match entity.Match) error {
	parsedContestID := uuid.MustParse(contestID)

	countryMap, err := r.GetCountryCodeToIDMap(ctx)
	if err != nil {
		return err
	}

	var c1ID, c2ID *uuid.UUID
	if match.Country1 != nil {
		if idStr, ok := countryMap[match.Country1.Code]; ok {
			id := uuid.MustParse(idStr)
			c1ID = &id
		}
	}
	if match.Country2 != nil {
		if idStr, ok := countryMap[match.Country2.Code]; ok {
			id := uuid.MustParse(idStr)
			c2ID = &id
		}
	}

	var c1g, c2g, c1p, c2p, c1cs, c2cs *int32
	if match.Country1Goals != nil {
		val := int32(*match.Country1Goals)
		c1g = &val
	}
	if match.Country2Goals != nil {
		val := int32(*match.Country2Goals)
		c2g = &val
	}
	if match.Country1Penalties != nil {
		val := int32(*match.Country1Penalties)
		c1p = &val
	}
	if match.Country2Penalties != nil {
		val := int32(*match.Country2Penalties)
		c2p = &val
	}
	if match.Country1ConductScore != nil {
		val := int32(*match.Country1ConductScore)
		c1cs = &val
	}
	if match.Country2ConductScore != nil {
		val := int32(*match.Country2ConductScore)
		c2cs = &val
	}

	var existingMatch model.Matches
	var isKnockout bool

	// 1. Find the existing match either by Round/Index or by Countries
	if match.Round > 0 && match.RoundIndex != nil {
		stmt := postgres.SELECT(table.Matches.AllColumns).FROM(table.Matches).WHERE(
			table.Matches.ContestID.EQ(postgres.UUID(parsedContestID)).
				AND(table.Matches.Round.EQ(postgres.Int32(int32(match.Round)))).
				AND(table.Matches.RoundIndex.EQ(postgres.Int32(int32(*match.RoundIndex)))),
		)

		if err := stmt.QueryContext(ctx, r.db, &existingMatch); err != nil {
			return fmt.Errorf("failed to find knockout match by round/index: %w", err)
		}
		isKnockout = true
	} else {
		// Lookup by countries & round to prevent double-encounter ambiguity
		if c1ID == nil || c2ID == nil {
			return errors.New("match must have both countries defined to update by countries")
		}

		stmt := postgres.SELECT(table.Matches.AllColumns).FROM(table.Matches).WHERE(
			table.Matches.ContestID.EQ(postgres.UUID(parsedContestID)).
				AND(table.Matches.Round.EQ(postgres.Int32(int32(match.Round)))).
				AND(
					(table.Matches.Country1ID.EQ(postgres.UUID(*c1ID)).AND(table.Matches.Country2ID.EQ(postgres.UUID(*c2ID)))).
						OR(table.Matches.Country1ID.EQ(postgres.UUID(*c2ID)).AND(table.Matches.Country2ID.EQ(postgres.UUID(*c1ID)))),
				),
		)

		if err := stmt.QueryContext(ctx, r.db, &existingMatch); err != nil {
			return fmt.Errorf("failed to find match by countries: %w", err)
		}
		isKnockout = existingMatch.Round > 0
	}

	// 2. Modify only the provided (non-nil) fields on the existing record
	updateModel := existingMatch
	var hasUpdates bool

	// Countries can only be updated for Knockout stage matches (Group stage pairings are static)
	if isKnockout {
		if c1ID != nil {
			updateModel.Country1ID = c1ID
			hasUpdates = true
		}
		if c2ID != nil {
			updateModel.Country2ID = c2ID
			hasUpdates = true
		}
	}

	if match.Country1Goals != nil {
		updateModel.Country1Goals = c1g
		hasUpdates = true
	}
	if match.Country2Goals != nil {
		updateModel.Country2Goals = c2g
		hasUpdates = true
	}
	if match.Country1ConductScore != nil {
		updateModel.Country1ConductScore = c1cs
		hasUpdates = true
	}
	if match.Country2ConductScore != nil {
		updateModel.Country2ConductScore = c2cs
		hasUpdates = true
	}

	// Penalties are only updated for Knockout stage matches (Group stage matches can end in draws)
	if isKnockout {
		if match.Country1Penalties != nil {
			updateModel.Country1Penalties = c1p
			hasUpdates = true
		}
		if match.Country2Penalties != nil {
			updateModel.Country2Penalties = c2p
			hasUpdates = true
		}
	}

	if !hasUpdates {
		return nil // Nothing to update
	}

	// 3. Perform the update and side effects in a single SQL transaction to guarantee consistency
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the match record
	updateStmt := table.Matches.UPDATE(
		table.Matches.Country1ID,
		table.Matches.Country2ID,
		table.Matches.Country1Goals,
		table.Matches.Country2Goals,
		table.Matches.Country1Penalties,
		table.Matches.Country2Penalties,
		table.Matches.Country1ConductScore,
		table.Matches.Country2ConductScore,
	).
		MODEL(updateModel).
		WHERE(table.Matches.ID.EQ(postgres.UUID(existingMatch.ID)))

	if _, err := updateStmt.ExecContext(ctx, tx); err != nil {
		return fmt.Errorf("failed to update match: %w", err)
	}

	// If this is a group stage match and a score has been recorded/updated, update the two countries' standings
	if !isKnockout && updateModel.Country1Goals != nil && updateModel.Country2Goals != nil {
		if existingMatch.Country1ID == nil || existingMatch.Country2ID == nil {
			return errors.New("cannot update standings for match with undefined countries")
		}
		c1ID := *existingMatch.Country1ID
		c2ID := *existingMatch.Country2ID

		var standing1, standing2 model.GroupStandings

		// Fetch current standings
		stmt1 := postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
			table.GroupStandings.ContestID.EQ(postgres.UUID(existingMatch.ContestID)).
				AND(table.GroupStandings.CountryID.EQ(postgres.UUID(c1ID))),
		)
		if err := stmt1.QueryContext(ctx, tx, &standing1); err != nil {
			return fmt.Errorf("failed to fetch standing for country1: %w", err)
		}

		stmt2 := postgres.SELECT(table.GroupStandings.AllColumns).FROM(table.GroupStandings).WHERE(
			table.GroupStandings.ContestID.EQ(postgres.UUID(existingMatch.ContestID)).
				AND(table.GroupStandings.CountryID.EQ(postgres.UUID(c2ID))),
		)
		if err := stmt2.QueryContext(ctx, tx, &standing2); err != nil {
			return fmt.Errorf("failed to fetch standing for country2: %w", err)
		}

		// 1. Subtract previous score contributions if a score was already recorded
		if existingMatch.Country1Goals != nil && existingMatch.Country2Goals != nil {
			oldG1 := *existingMatch.Country1Goals
			oldG2 := *existingMatch.Country2Goals
			oldCs1 := int32(0)
			if existingMatch.Country1ConductScore != nil {
				oldCs1 = *existingMatch.Country1ConductScore
			}
			oldCs2 := int32(0)
			if existingMatch.Country2ConductScore != nil {
				oldCs2 = *existingMatch.Country2ConductScore
			}

			// Revert Country 1 statistics
			standing1.Gf -= oldG1
			standing1.Ga -= oldG2
			standing1.Gd = standing1.Gf - standing1.Ga
			standing1.Cs -= oldCs1

			// Revert Country 2 statistics
			standing2.Gf -= oldG2
			standing2.Ga -= oldG1
			standing2.Gd = standing2.Gf - standing2.Ga
			standing2.Cs -= oldCs2

			// Revert wins/draws/losses/points
			if oldG1 > oldG2 {
				standing1.Wins--
				standing1.Points -= 3
				standing2.Losses--
			} else if oldG1 == oldG2 {
				standing1.Draws--
				standing1.Points -= 1
				standing2.Draws--
				standing2.Points -= 1
			} else {
				standing1.Losses--
				standing2.Wins--
				standing2.Points -= 3
			}
		}

		// 2. Add the new score contributions
		newG1 := *updateModel.Country1Goals
		newG2 := *updateModel.Country2Goals
		newCs1 := int32(0)
		if updateModel.Country1ConductScore != nil {
			newCs1 = *updateModel.Country1ConductScore
		}
		newCs2 := int32(0)
		if updateModel.Country2ConductScore != nil {
			newCs2 = *updateModel.Country2ConductScore
		}

		// Apply Country 1 statistics
		standing1.Gf += newG1
		standing1.Ga += newG2
		standing1.Gd = standing1.Gf - standing1.Ga
		standing1.Cs += newCs1

		// Apply Country 2 statistics
		standing2.Gf += newG2
		standing2.Ga += newG1
		standing2.Gd = standing2.Gf - standing2.Ga
		standing2.Cs += newCs2

		// Apply wins/draws/losses/points
		if newG1 > newG2 {
			standing1.Wins++
			standing1.Points += 3
			standing2.Losses++
		} else if newG1 == newG2 {
			standing1.Draws++
			standing1.Points += 1
			standing2.Draws++
			standing2.Points += 1
		} else {
			standing1.Losses++
			standing2.Wins++
			standing2.Points += 3
		}

		// 3. Persist the updated standings for both countries
		updStandingStmt1 := table.GroupStandings.UPDATE(
			table.GroupStandings.Wins,
			table.GroupStandings.Draws,
			table.GroupStandings.Losses,
			table.GroupStandings.Gf,
			table.GroupStandings.Ga,
			table.GroupStandings.Gd,
			table.GroupStandings.Points,
			table.GroupStandings.Cs,
		).
			MODEL(standing1).
			WHERE(table.GroupStandings.ContestID.EQ(postgres.UUID(existingMatch.ContestID)).
				AND(table.GroupStandings.CountryID.EQ(postgres.UUID(c1ID))))

		if _, err := updStandingStmt1.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to update standing for country1: %w", err)
		}

		updStandingStmt2 := table.GroupStandings.UPDATE(
			table.GroupStandings.Wins,
			table.GroupStandings.Draws,
			table.GroupStandings.Losses,
			table.GroupStandings.Gf,
			table.GroupStandings.Ga,
			table.GroupStandings.Gd,
			table.GroupStandings.Points,
			table.GroupStandings.Cs,
		).
			MODEL(standing2).
			WHERE(table.GroupStandings.ContestID.EQ(postgres.UUID(existingMatch.ContestID)).
				AND(table.GroupStandings.CountryID.EQ(postgres.UUID(c2ID))))

		if _, err := updStandingStmt2.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to update standing for country2: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
