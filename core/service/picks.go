package service

import (
	"context"
	"errors"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type PicksService struct {
	repo ports.PicksRepository
}

var _ ports.PicksService = (*PicksService)(nil)

func NewPicksService(repo ports.PicksRepository) *PicksService {
	return &PicksService{repo: repo}
}

func (s *PicksService) ListGroupPicks(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return nil, nil, err
	}
	if contest == nil {
		return nil, nil, errors.New("contest not found")
	}

	picks, err := s.repo.ListGroupPicks(ctx, userID, contest.ID)
	if err != nil {
		return nil, nil, err
	}

	standings, err := s.repo.ListGroupStandings(ctx, contest.ID)
	if err != nil {
		return nil, nil, err
	}

	return picks, standings, nil
}

func (s *PicksService) CreateGroupPicks(ctx context.Context, contestSlug string, pick entity.GroupPick) error {
	return nil
}

func (s *PicksService) ListKnockoutPicks(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error) {
	return nil, nil
}

func (s *PicksService) CreateKnockoutPicks(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error {
	return nil
}
