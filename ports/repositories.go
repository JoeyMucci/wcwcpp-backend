package ports

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
)

type TokenValidator interface {
	ValidateGoogleToken(ctx context.Context, token string) (email string, err error)
}

type Search interface {
	GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error)
	GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error)
	GetCountryCodeToIDMap(ctx context.Context) (map[string]string, error)
}

type ContestRepository interface {
	ListContests(ctx context.Context) ([]entity.Contest, error)
	CreateContest(ctx context.Context, contest *entity.Contest) error
	CreateCountries(ctx context.Context, countries []entity.Country) error
	CreateMatches(ctx context.Context, contestID string, matches []entity.Match) error
	CreateGroupStandings(ctx context.Context, contestID string, groups []entity.Group) error
	CreateSubcontest(ctx context.Context, subcontest *entity.Subcontest) error
	JoinSubcontest(ctx context.Context, subcontestID string, userID string) error
	GetSubcontestByJoinCode(ctx context.Context, joinCode string) (*entity.Subcontest, error)
	ListSubcontests(ctx context.Context, contestID string, userID string) ([]entity.Subcontest, error)
	DeleteSubcontest(ctx context.Context, subcontestID string) error
	Search
}

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	CreateUser(ctx context.Context, email string, username string) (*entity.User, error)
	CountUsers(ctx context.Context) (int64, error)
	DeleteUser(ctx context.Context, userID string) error
}

type LeaderboardRepository interface {
	Leaderboard(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	Subleaderboard(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	HasSubcontestAccess(ctx context.Context, userID string, subcontestSlug string) (bool, error)
	Search
}

type StandingsRepository interface {
	IncrementUserGroupScore(ctx context.Context, userID string, contestID string, score int)
	IncrementUserKnockoutScore(ctx context.Context, userID string, contestID string, score int)
}

type PicksRepository interface {
	ListGroupPicks(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error)
	ListGroupStandings(ctx context.Context, contestID string) ([]entity.GroupStanding, error)
	CreateGroupPicks(ctx context.Context, userID string, contestID string, picks []entity.GroupPick) error
	Search
}
