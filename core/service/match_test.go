package service

import (
	"context"
	"errors"
	"testing"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
	"github.com/stretchr/testify/require"
)

type mockMatchContestRepository struct {
	ports.ContestRepository
	getContestBySlugFunc func(ctx context.Context, slug string) (*entity.Contest, error)
	listGroupMatchesFunc func(ctx context.Context, contestID string, letter string) ([]entity.Match, error)
	listKnockoutMatchesFunc func(ctx context.Context, contestID string) ([]entity.Match, error)
	updateMatchFunc      func(ctx context.Context, contestID string, match entity.Match) error
}

func (m *mockMatchContestRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	if m.getContestBySlugFunc != nil {
		return m.getContestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockMatchContestRepository) ListGroupMatches(ctx context.Context, contestID string, letter string) ([]entity.Match, error) {
	if m.listGroupMatchesFunc != nil {
		return m.listGroupMatchesFunc(ctx, contestID, letter)
	}
	return nil, nil
}

func (m *mockMatchContestRepository) ListKnockoutMatches(ctx context.Context, contestID string) ([]entity.Match, error) {
	if m.listKnockoutMatchesFunc != nil {
		return m.listKnockoutMatchesFunc(ctx, contestID)
	}
	return nil, nil
}

func (m *mockMatchContestRepository) UpdateMatch(ctx context.Context, contestID string, match entity.Match) error {
	if m.updateMatchFunc != nil {
		return m.updateMatchFunc(ctx, contestID, match)
	}
	return nil
}

func TestMatchService_ListGroupMatches(t *testing.T) {
	t.Run("should list group matches on success", func(t *testing.T) {
		expectedMatches := []entity.Match{
			{
				Round:    0,
				Country1: &entity.Country{Code: "USA", FullName: "United States"},
				Country2: &entity.Country{Code: "MEX", FullName: "Mexico"},
			},
		}

		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				require.Equal(t, "world-cup-2026", slug)
				return &entity.Contest{ID: "contest-123"}, nil
			},
			listGroupMatchesFunc: func(ctx context.Context, contestID string, letter string) ([]entity.Match, error) {
				require.Equal(t, "contest-123", contestID)
				require.Equal(t, "A", letter)
				return expectedMatches, nil
			},
		}

		svc := NewMatchService(mockRepo)
		matches, err := svc.ListGroupMatches(context.Background(), "world-cup-2026", "A")
		require.NoError(t, err)
		require.Equal(t, expectedMatches, matches)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewMatchService(mockRepo)
		_, err := svc.ListGroupMatches(context.Background(), "world-cup-2026", "A")
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})

	t.Run("should propagate database error", func(t *testing.T) {
		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("db connection failure")
			},
		}

		svc := NewMatchService(mockRepo)
		_, err := svc.ListGroupMatches(context.Background(), "world-cup-2026", "A")
		require.Error(t, err)
		require.Equal(t, "db connection failure", err.Error())
	})
}

func TestMatchService_ListKnockoutMatches(t *testing.T) {
	t.Run("should list knockout matches on success", func(t *testing.T) {
		expectedMatches := []entity.Match{
			{
				Round: 1,
			},
		}

		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-123"}, nil
			},
			listKnockoutMatchesFunc: func(ctx context.Context, contestID string) ([]entity.Match, error) {
				require.Equal(t, "contest-123", contestID)
				return expectedMatches, nil
			},
		}

		svc := NewMatchService(mockRepo)
		matches, err := svc.ListKnockoutMatches(context.Background(), "world-cup-2026")
		require.NoError(t, err)
		require.Equal(t, expectedMatches, matches)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewMatchService(mockRepo)
		_, err := svc.ListKnockoutMatches(context.Background(), "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})
}

func TestMatchService_CreateMatch(t *testing.T) {
	t.Run("should update/create match on success", func(t *testing.T) {
		goals1, goals2 := 1, 2
		inputMatch := entity.Match{
			Country1:      &entity.Country{Code: "CAN"},
			Country2:      &entity.Country{Code: "ARG"},
			Country1Goals: &goals1,
			Country2Goals: &goals2,
		}

		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-123"}, nil
			},
			updateMatchFunc: func(ctx context.Context, contestID string, match entity.Match) error {
				require.Equal(t, "contest-123", contestID)
				require.Equal(t, "CAN", match.Country1.Code)
				require.Equal(t, 1, *match.Country1Goals)
				return nil
			},
		}

		svc := NewMatchService(mockRepo)
		err := svc.CreateMatch(context.Background(), "world-cup-2026", inputMatch)
		require.NoError(t, err)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		mockRepo := &mockMatchContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewMatchService(mockRepo)
		err := svc.CreateMatch(context.Background(), "world-cup-2026", entity.Match{})
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})
}
