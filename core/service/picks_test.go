package service

import (
	"context"
	"errors"
	"testing"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
	"github.com/stretchr/testify/require"
)

type mockPicksRepository struct {
	ports.PicksRepository
	getContestBySlugFunc   func(ctx context.Context, slug string) (*entity.Contest, error)
	getSubcontestBySlugFunc func(ctx context.Context, slug string) (*entity.Subcontest, error)
	listGroupPicksFunc     func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error)
	listGroupStandingsFunc func(ctx context.Context, contestID string) ([]entity.GroupStanding, error)
}

func (m *mockPicksRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	if m.getContestBySlugFunc != nil {
		return m.getContestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockPicksRepository) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	if m.getSubcontestBySlugFunc != nil {
		return m.getSubcontestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockPicksRepository) ListGroupPicks(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
	if m.listGroupPicksFunc != nil {
		return m.listGroupPicksFunc(ctx, userID, contestID)
	}
	return nil, nil
}

func (m *mockPicksRepository) ListGroupStandings(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
	if m.listGroupStandingsFunc != nil {
		return m.listGroupStandingsFunc(ctx, contestID)
	}
	return nil, nil
}

func TestPicksService_ListGroupPicks(t *testing.T) {
	t.Run("should return picks and standings on success", func(t *testing.T) {
		expectedPicks := []entity.GroupPick{
			{Letter: "A", Entries: []entity.GroupPickEntry{
				{Country: entity.Country{Code: "USA"}, Place: 1},
				{Country: entity.Country{Code: "MEX"}, Place: 2},
				{Country: entity.Country{Code: "CAN"}, Place: 3},
				{Country: entity.Country{Code: "ARG"}, Place: 4},
			}},
		}
		expectedStandings := []entity.GroupStanding{
			{Country: entity.Country{Code: "USA"}, Letter: "A", Points: 3},
		}

		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listGroupPicksFunc: func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
				require.Equal(t, "user-1", userID)
				require.Equal(t, "contest-1", contestID)
				return expectedPicks, nil
			},
			listGroupStandingsFunc: func(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
				require.Equal(t, "contest-1", contestID)
				return expectedStandings, nil
			},
		}

		svc := NewPicksService(repo)
		picks, standings, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.NoError(t, err)
		require.Equal(t, expectedPicks, picks)
		require.Equal(t, expectedStandings, standings)
	})

	t.Run("should return error if contest not found (nil)", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "nonexistent")
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})

	t.Run("should propagate error from contest lookup", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "db error", err.Error())
	})

	t.Run("should propagate error from ListGroupPicks repo", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listGroupPicksFunc: func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
				return nil, errors.New("picks query failed")
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "picks query failed", err.Error())
	})

	t.Run("should propagate error from ListGroupStandings repo", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listGroupPicksFunc: func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
				return nil, nil
			},
			listGroupStandingsFunc: func(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
				return nil, errors.New("standings query failed")
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "standings query failed", err.Error())
	})
}
