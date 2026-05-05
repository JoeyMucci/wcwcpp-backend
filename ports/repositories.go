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
	ListContests(ctx context.Context) ([]entity.Contest, error)
	CreateContest(ctx context.Context, contest *entity.Contest) error
	CreateCountries(ctx context.Context, countries []entity.Country) error
	CreateMatches(ctx context.Context, contestID string, matches []entity.Match) error
	GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error)
	CreateSubcontest(ctx context.Context, subcontest *entity.Subcontest) error
	JoinSubcontest(ctx context.Context, subcontestID string, userID string) error
	GetSubcontestByJoinCode(ctx context.Context, joinCode string) (*entity.Subcontest, error)
	GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error)
	ListSubcontests(ctx context.Context, contestID string, userID string) ([]entity.Subcontest, error)
	DeleteSubcontest(ctx context.Context, subcontestID string) error
}
