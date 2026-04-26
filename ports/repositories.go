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

type ContestRepository interface {
	CreateContest(ctx context.Context, contest *entity.Contest) error
	CreateCountries(ctx context.Context, countries []entity.Country) error
	CreateMatches(ctx context.Context, contestID string, matches []entity.Match) error
}
