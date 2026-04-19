package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type PicksService struct{}

var _ ports.PicksService = (*PicksService)(nil)

func NewPicksService() *PicksService {
	return &PicksService{}
}

func (s *PicksService) ListGroupPicks(ctx context.Context, contestSlug string) ([]entity.GroupPick, error) {
	return nil, nil
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
