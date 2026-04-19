package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestService struct{}

var _ ports.ContestService = (*ContestService)(nil)

func NewContestService() *ContestService {
	return &ContestService{}
}

func (s *ContestService) ListContests(ctx context.Context) ([]entity.Contest, error) {
	return nil, nil
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
