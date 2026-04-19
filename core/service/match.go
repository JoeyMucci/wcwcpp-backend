package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type MatchService struct{}

var _ ports.MatchService = (*MatchService)(nil)

func NewMatchService() *MatchService {
	return &MatchService{}
}

func (s *MatchService) ListGroupMatches(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
	return nil, nil
}

func (s *MatchService) ListKnockoutMatches(ctx context.Context, contestSlug string) ([]entity.Match, error) {
	return nil, nil
}

func (s *MatchService) CreateMatch(ctx context.Context, contestSlug string, match entity.Match) error {
	return nil
}
