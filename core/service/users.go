package service

import (
	"context"

	"github.com/joey/wcwcpp-backend/ports"
)

type UsersService struct{}

var _ ports.UsersService = (*UsersService)(nil)

func NewUsersService() *UsersService {
	return &UsersService{}
}

func (s *UsersService) CountUsers(ctx context.Context) (int64, error) {
	return 0, nil
}
