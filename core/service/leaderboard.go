package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type LeaderboardService struct{}

var _ ports.LeaderboardService = (*LeaderboardService)(nil)

func NewLeaderboardService() *LeaderboardService {
	return &LeaderboardService{}
}

func (s *LeaderboardService) Leaderboard(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return nil, nil
}

func (s *LeaderboardService) Subleaderboard(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return nil, nil
}
