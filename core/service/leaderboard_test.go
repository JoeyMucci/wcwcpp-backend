package service

import (
	"context"
	"errors"
	"testing"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLeaderboardRepository struct {
	getContestBySlugFunc    func(ctx context.Context, slug string) (*entity.Contest, error)
	getSubcontestBySlugFunc func(ctx context.Context, slug string) (*entity.Subcontest, error)
	leaderboardFunc         func(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	subleaderboardFunc      func(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	hasSubcontestAccessFunc func(ctx context.Context, userID string, subcontestSlug string) (bool, error)
}

func (m *mockLeaderboardRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	return m.getContestBySlugFunc(ctx, slug)
}

func (m *mockLeaderboardRepository) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	return m.getSubcontestBySlugFunc(ctx, slug)
}

func (m *mockLeaderboardRepository) Leaderboard(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.leaderboardFunc(ctx, contestID, limit, offset)
}

func (m *mockLeaderboardRepository) Subleaderboard(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.subleaderboardFunc(ctx, subcontestID, limit, offset)
}

func (m *mockLeaderboardRepository) HasSubcontestAccess(ctx context.Context, userID string, subcontestSlug string) (bool, error) {
	return m.hasSubcontestAccessFunc(ctx, userID, subcontestSlug)
}

func TestLeaderboardService_Leaderboard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockLeaderboardRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				assert.Equal(t, "world-cup-2026", slug)
				return &entity.Contest{ID: "contest-uuid", Title: "World Cup 2026", Slug: "world-cup-2026"}, nil
			},
			leaderboardFunc: func(ctx context.Context, contestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				assert.Equal(t, "contest-uuid", contestID)
				assert.Equal(t, int32(10), limit)
				assert.Equal(t, int32(0), offset)
				return map[string][]entity.LeaderboardEntry{
					"group": {
						{Name: "Alice", Score: 10},
					},
				}, nil
			},
		}

		svc := NewLeaderboardService(repo)
		res, err := svc.Leaderboard(context.Background(), "world-cup-2026", 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, int64(10), res["group"][0].Score)
	})

	t.Run("contest not found", func(t *testing.T) {
		repo := &mockLeaderboardRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewLeaderboardService(repo)
		_, err := svc.Leaderboard(context.Background(), "unknown", 10, 0)
		require.Error(t, err)
		assert.Equal(t, "contest not found", err.Error())
	})

	t.Run("db error on slug lookup", func(t *testing.T) {
		repo := &mockLeaderboardRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("db connection failure")
			},
		}

		svc := NewLeaderboardService(repo)
		_, err := svc.Leaderboard(context.Background(), "world-cup-2026", 10, 0)
		require.Error(t, err)
		assert.Equal(t, "db connection failure", err.Error())
	})
}

func TestLeaderboardService_Subleaderboard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockLeaderboardRepository{
			getSubcontestBySlugFunc: func(ctx context.Context, slug string) (*entity.Subcontest, error) {
				assert.Equal(t, "alice-subcontest", slug)
				return &entity.Subcontest{ID: "subcontest-uuid", ContestID: "contest-uuid", Title: "Alice Subcontest", Slug: "alice-subcontest"}, nil
			},
			subleaderboardFunc: func(ctx context.Context, subcontestID string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				assert.Equal(t, "subcontest-uuid", subcontestID)
				assert.Equal(t, int32(5), limit)
				assert.Equal(t, int32(2), offset)
				return map[string][]entity.LeaderboardEntry{
					"knockout": {
						{Name: "Bob", Score: 12},
					},
				}, nil
			},
		}

		svc := NewLeaderboardService(repo)
		res, err := svc.Subleaderboard(context.Background(), "alice-subcontest", 5, 2)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, int64(12), res["knockout"][0].Score)
	})

	t.Run("subcontest not found", func(t *testing.T) {
		repo := &mockLeaderboardRepository{
			getSubcontestBySlugFunc: func(ctx context.Context, slug string) (*entity.Subcontest, error) {
				return nil, nil
			},
		}

		svc := NewLeaderboardService(repo)
		_, err := svc.Subleaderboard(context.Background(), "unknown", 10, 0)
		require.Error(t, err)
		assert.Equal(t, "subcontest not found", err.Error())
	})
}
