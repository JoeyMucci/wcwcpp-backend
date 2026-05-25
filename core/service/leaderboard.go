package service

import (
	"context"
	"errors"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type LeaderboardService struct {
	repo ports.LeaderboardRepository
}

var _ ports.LeaderboardService = (*LeaderboardService)(nil)

func NewLeaderboardService(repo ports.LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo: repo}
}

func (s *LeaderboardService) Leaderboard(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	contest, err := s.repo.GetContestBySlug(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	if contest == nil {
		return nil, errors.New("contest not found")
	}

	return s.repo.Leaderboard(ctx, contest.ID, limit, offset)
}

func (s *LeaderboardService) Subleaderboard(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	hasAccess, err := s.repo.HasSubcontestAccess(ctx, userID, subcontestSlug)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, errors.New("permission denied: no access to subcontest")
	}

	subcontest, err := s.repo.GetSubcontestBySlug(ctx, subcontestSlug)
	if err != nil {
		return nil, err
	}
	if subcontest == nil {
		return nil, errors.New("subcontest not found")
	}

	return s.repo.Subleaderboard(ctx, subcontest.ID, limit, offset)
}
