package ports

import (
	"context"

	"github.com/joey/wcwcpp-backend/core/entity"
)

type ContestService interface {
	ListContests(ctx context.Context) ([]entity.Contest, error)
	CreateContest(ctx context.Context, contest entity.Contest) error
	ListSubcontests(ctx context.Context, userID string, contestSlug string) ([]entity.Subcontest, error)
	CreateSubcontest(ctx context.Context, userID string, contestSlug string, title string, selfJoin bool) (string, error)
	DeleteSubcontest(ctx context.Context, userID string, subcontestSlug string) error
	JoinSubcontest(ctx context.Context, userID string, joinCode string) error
}

type LeaderboardService interface {
	Leaderboard(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
	Subleaderboard(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
}

type MatchService interface {
	ListGroupMatches(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error)
	ListKnockoutMatches(ctx context.Context, contestSlug string) ([]entity.Match, error)
	CreateMatch(ctx context.Context, contestSlug string, match entity.Match) error
}

type PicksService interface {
	ListGroupPicks(ctx context.Context, contestSlug string) ([]entity.GroupPick, error)
	CreateGroupPicks(ctx context.Context, contestSlug string, pick entity.GroupPick) error
	ListKnockoutPicks(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error)
	CreateKnockoutPicks(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error
}

type UsersService interface {
	CountUsers(ctx context.Context) (int64, error)
}

type AuthService interface {
	Login(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error)
}
