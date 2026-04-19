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

func (s *LeaderboardService) Leaderboard(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
	return nil, "", nil
}

func (s *LeaderboardService) Subleaderboard(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
	return nil, "", nil
}
