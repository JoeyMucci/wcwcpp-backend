package service

import (
	"context"
	"errors"
	"time"

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

func (s *PicksService) CreateGroupPicks(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return err
	}
	if contest == nil {
		return errors.New("contest not found")
	}

	// Lock checks
	now := time.Now()
	if !contest.GroupUnlockDate.IsZero() && now.Before(contest.GroupUnlockDate) {
		return errors.New("group stage picks are locked")
	}
	if !contest.GroupLockDate.IsZero() && now.After(contest.GroupLockDate) {
		return errors.New("group stage picks are locked")
	}

	err = s.repo.CreateGroupPicks(ctx, userID, contest.ID, picks)
	if err != nil {
		return err
	}

	return nil
}

func (s *PicksService) ListKnockoutPicks(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return entity.KnockoutPick{}, entity.KnockoutPick{}, err
	}
	if contest == nil {
		return entity.KnockoutPick{}, entity.KnockoutPick{}, errors.New("contest not found")
	}

	pick, err := s.repo.ListKnockoutPicks(ctx, userID, contest.ID)
	if err != nil {
		return entity.KnockoutPick{}, entity.KnockoutPick{}, err
	}

	result, err := s.repo.ListKnockoutResults(ctx, contest.ID)
	if err != nil {
		return entity.KnockoutPick{}, entity.KnockoutPick{}, err
	}

	return pick, result, nil
}

func (s *PicksService) CreateKnockoutPicks(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return err
	}
	if contest == nil {
		return errors.New("contest not found")
	}

	// Lock checks
	now := time.Now()
	if !contest.KnockoutUnlockDate.IsZero() && now.Before(contest.KnockoutUnlockDate) {
		return errors.New("knockout stage picks are locked")
	}
	if !contest.KnockoutLockDate.IsZero() && now.After(contest.KnockoutLockDate) {
		return errors.New("knockout stage picks are locked")
	}

	return s.repo.CreateKnockoutPicks(ctx, userID, contest.ID, pick)
}
