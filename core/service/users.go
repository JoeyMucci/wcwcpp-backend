package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/ports"
)

type UsersService struct {
	repo ports.UserRepository
}

var _ ports.UsersService = (*UsersService)(nil)

func NewUsersService(repo ports.UserRepository) *UsersService {
	return &UsersService{repo: repo}
}

func (s *UsersService) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.CountUsers(ctx)
}

func (s *UsersService) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.DeleteUser(ctx, userID)
}
