package ports

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	CreateUser(ctx context.Context, email string, username string) (*entity.User, error)
}

type TokenValidator interface {
	ValidateGoogleToken(ctx context.Context, token string) (email string, err error)
}
