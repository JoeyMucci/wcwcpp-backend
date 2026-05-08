package service

import (
	"context"

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
	return s.repo.Leaderboard(ctx, contestSlug, limit, offset)
}

func (s *LeaderboardService) Subleaderboard(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {

	return s.repo.Subleaderboard(ctx, subcontestSlug, limit, offset)
}
