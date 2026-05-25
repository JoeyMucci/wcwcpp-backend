package service

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/gosimple/slug"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestService struct {
	repo ports.ContestRepository
}

var _ ports.ContestService = (*ContestService)(nil)

func NewContestService(repo ports.ContestRepository) *ContestService {
	return &ContestService{repo: repo}
}

func (s *ContestService) ListContests(ctx context.Context) ([]entity.Contest, error) {
	return s.repo.ListContests(ctx)
}

func (s *ContestService) CreateContest(ctx context.Context, contest entity.Contest) error {
	// 1. Create Contest
	if contest.Slug == "" {
		contest.Slug = slug.Make(contest.Title)
	}
	if err := s.repo.CreateContest(ctx, &contest); err != nil {
		return err
	}

	// 2. Collect and Create Countries
	var allCountries []entity.Country
	for _, group := range contest.Groups {
		allCountries = append(allCountries, group.Countries...)
	}
	if err := s.repo.CreateCountries(ctx, allCountries); err != nil {
		return err
	}

	// 3. Generate Matches
	var matches []entity.Match
	for _, group := range contest.Groups {
		for i := 0; i < len(group.Countries); i++ {
			for j := i + 1; j < len(group.Countries); j++ {
				c1 := group.Countries[i]
				c2 := group.Countries[j]
				matches = append(matches, entity.Match{
					Country1: &c1,
					Country2: &c2,
					Round:    0,
				})
			}
		}
	}

	knockouts := []struct {
		round int
		count int
	}{
		{1, 16}, {2, 8}, {3, 4}, {4, 2}, {5, 2},
	}

	for _, k := range knockouts {
		for i := range k.count {
			idx := i
			matches = append(matches, entity.Match{
				Round:      k.round,
				RoundIndex: &idx,
			})
		}
	}

	// 4. Create Matches
	if err := s.repo.CreateMatches(ctx, contest.ID, matches); err != nil {
		return err
	}

	// 5. Seed group_standings with zero values for each country in each group
	if err := s.repo.CreateGroupStandings(ctx, contest.ID, contest.Groups); err != nil {
		return err
	}

	return nil
}

func (s *ContestService) ListSubcontests(ctx context.Context, userID string, contestSlug string) ([]entity.Subcontest, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	if contest == nil {
		return nil, errors.New("contest not found")
	}

	return s.repo.ListSubcontests(ctx, contest.ID, userID)
}

func (s *ContestService) CreateSubcontest(ctx context.Context, userID string, contestSlug string, title string, selfJoin bool) (string, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return "", err
	}
	if contest == nil {
		return "", errors.New("contest not found")
	}

	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ0123456789"
	joinCodeBytes := make([]byte, 8)
	if _, err := rand.Read(joinCodeBytes); err != nil {
		return "", err
	}
	for i, b := range joinCodeBytes {
		joinCodeBytes[i] = charset[b%byte(len(charset))]
	}
	joinCode := string(joinCodeBytes)

	sub := &entity.Subcontest{
		ContestID: contest.ID,
		UserID:    userID,
		Title:     title,
		Slug:      slug.Make(title),
		JoinCode:  joinCode,
	}

	if err := s.repo.CreateSubcontest(ctx, sub); err != nil {
		return "", err
	}

	if selfJoin {
		if err := s.repo.JoinSubcontest(ctx, sub.ID, userID); err != nil {
			return "", err
		}
	}

	return joinCode, nil
}

func (s *ContestService) DeleteSubcontest(ctx context.Context, userID string, subcontestSlug string) error {
	sub, err := s.repo.GetSubcontestBySlug(ctx, subcontestSlug)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("subcontest not found")
	}

	if sub.UserID != userID {
		return errors.New("not owner")
	}

	return s.repo.DeleteSubcontest(ctx, sub.ID)
}

func (s *ContestService) JoinSubcontest(ctx context.Context, userID string, joinCode string) error {
	sub, err := s.repo.GetSubcontestByJoinCode(ctx, joinCode)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("invalid join code")
	}

	return s.repo.JoinSubcontest(ctx, sub.ID, userID)
}
