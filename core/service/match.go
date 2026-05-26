package service

import (
	"context"
	"errors"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type MatchService struct {
	repo ports.ContestRepository
}

var _ ports.MatchService = (*MatchService)(nil)

func NewMatchService(repo ports.ContestRepository) *MatchService {
	return &MatchService{repo: repo}
}

func (s *MatchService) ListGroupMatches(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	if contest == nil {
		return nil, errors.New("contest not found")
	}

	return s.repo.ListGroupMatches(ctx, contest.ID, letter)
}

func (s *MatchService) ListKnockoutMatches(ctx context.Context, contestSlug string) ([]entity.Match, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	if contest == nil {
		return nil, errors.New("contest not found")
	}

	return s.repo.ListKnockoutMatches(ctx, contest.ID)
}

func (s *MatchService) CreateMatch(ctx context.Context, contestSlug string, match entity.Match) error {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return err
	}
	if contest == nil {
		return errors.New("contest not found")
	}

	return s.repo.UpdateMatch(ctx, contest.ID, match)
}
