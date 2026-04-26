package service

import (
	"context"

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
	return nil, nil
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

	return nil
}

func (s *ContestService) ListSubcontests(ctx context.Context, contestSlug string) ([]entity.Contest, error) {
	return nil, nil
}

func (s *ContestService) CreateSubcontest(ctx context.Context, contestSlug string, title string) (string, error) {
	return "", nil
}

func (s *ContestService) DeleteSubcontest(ctx context.Context, subcontestSlug string) error {
	return nil
}
