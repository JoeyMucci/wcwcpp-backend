package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMatchService struct {
	ports.MatchService
	listGroupMatchesFunc    func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error)
	listKnockoutMatchesFunc func(ctx context.Context, contestSlug string) ([]entity.Match, error)
	createMatchFunc         func(ctx context.Context, contestSlug string, match entity.Match) error
}

func (m *mockMatchService) ListGroupMatches(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
	return m.listGroupMatchesFunc(ctx, contestSlug, letter)
}

func (m *mockMatchService) ListKnockoutMatches(ctx context.Context, contestSlug string) ([]entity.Match, error) {
	return m.listKnockoutMatchesFunc(ctx, contestSlug)
}

func (m *mockMatchService) CreateMatch(ctx context.Context, contestSlug string, match entity.Match) error {
	return m.createMatchFunc(ctx, contestSlug, match)
}

func TestMatchHandler_ListGroupMatches(t *testing.T) {
	t.Run("success with mapped fields", func(t *testing.T) {
		goals1, goals2 := 2, 1
		cs1, cs2 := 3, 1
		mockedMatches := []entity.Match{
			{
				Country1:             &entity.Country{Code: "USA", FullName: "United States"},
				Country2:             &entity.Country{Code: "MEX", FullName: "Mexico"},
				Country1Goals:        &goals1,
				Country2Goals:        &goals2,
				Country1ConductScore: &cs1,
				Country2ConductScore: &cs2,
				Round:                0,
			},
		}

		svc := &mockMatchService{
			listGroupMatchesFunc: func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
				assert.Equal(t, "world-cup-2026", contestSlug)
				assert.Equal(t, "A", letter)
				return mockedMatches, nil
			},
		}
		h := NewMatchHandler(svc)

		resp, err := h.ListGroupMatches(context.Background(), connect.NewRequest(&v1.ListGroupMatchesRequest{
			ContestSlug: "world-cup-2026",
			Letter:      "A",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Msg.Matches, 1)

		protoMatch := resp.Msg.Matches[0]
		assert.Equal(t, "USA", protoMatch.Country1.Code)
		assert.Equal(t, "MEX", protoMatch.Country2.Code)
		assert.Equal(t, int64(2), *protoMatch.Country1Goals)
		assert.Equal(t, int64(1), *protoMatch.Country2Goals)
		assert.Equal(t, int64(3), *protoMatch.Country1ConductScore)
		assert.Equal(t, int64(1), *protoMatch.Country2ConductScore)
		assert.Nil(t, protoMatch.Country1Penalties)
	})

	t.Run("service error propagation", func(t *testing.T) {
		svc := &mockMatchService{
			listGroupMatchesFunc: func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
				return nil, errors.New("something went wrong")
			},
		}
		h := NewMatchHandler(svc)

		_, err := h.ListGroupMatches(context.Background(), connect.NewRequest(&v1.ListGroupMatchesRequest{
			ContestSlug: "world-cup-2026",
			Letter:      "A",
		}))
		require.Error(t, err)
		assert.Equal(t, "something went wrong", err.Error())
	})
}

func TestMatchHandler_ListKnockoutMatches(t *testing.T) {
	t.Run("success with mapped fields", func(t *testing.T) {
		goals1, goals2 := 1, 1
		penalties1, penalties2 := 5, 4
		mockedMatches := []entity.Match{
			{
				Country1:          &entity.Country{Code: "FRA", FullName: "France"},
				Country2:          &entity.Country{Code: "GER", FullName: "Germany"},
				Country1Goals:     &goals1,
				Country2Goals:     &goals2,
				Country1Penalties: &penalties1,
				Country2Penalties: &penalties2,
				Round:             1,
			},
		}

		svc := &mockMatchService{
			listKnockoutMatchesFunc: func(ctx context.Context, contestSlug string) ([]entity.Match, error) {
				assert.Equal(t, "world-cup-2026", contestSlug)
				return mockedMatches, nil
			},
		}
		h := NewMatchHandler(svc)

		resp, err := h.ListKnockoutMatches(context.Background(), connect.NewRequest(&v1.ListKnockoutMatchesRequest{
			ContestSlug: "world-cup-2026",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Msg.Matches, 1)

		protoMatch := resp.Msg.Matches[0]
		assert.Equal(t, "FRA", protoMatch.Country1.Code)
		assert.Equal(t, "GER", protoMatch.Country2.Code)
		assert.Equal(t, int64(1), *protoMatch.Country1Goals)
		assert.Equal(t, int64(1), *protoMatch.Country2Goals)
		assert.Equal(t, int64(5), *protoMatch.Country1Penalties)
		assert.Equal(t, int64(4), *protoMatch.Country2Penalties)
	})

	t.Run("service error propagation", func(t *testing.T) {
		svc := &mockMatchService{
			listKnockoutMatchesFunc: func(ctx context.Context, contestSlug string) ([]entity.Match, error) {
				return nil, errors.New("knockout service error")
			},
		}
		h := NewMatchHandler(svc)

		_, err := h.ListKnockoutMatches(context.Background(), connect.NewRequest(&v1.ListKnockoutMatchesRequest{
			ContestSlug: "world-cup-2026",
		}))
		require.Error(t, err)
		assert.Equal(t, "knockout service error", err.Error())
	})
}

func generateMatchTestToken(secret, userID, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestMatchHandler_CreateMatch(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_secret")
	t.Setenv("SUPERADMIN_EMAILS", "super1@example.com")

	superToken := generateMatchTestToken("test_secret", "user1", "super1@example.com")
	normalToken := generateMatchTestToken("test_secret", "user2", "normal@example.com")

	t.Run("success with fully mapped request for superadmin", func(t *testing.T) {
		var capturedMatch entity.Match
		svc := &mockMatchService{
			createMatchFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				assert.Equal(t, "world-cup-2026", contestSlug)
				capturedMatch = match
				return nil
			},
		}
		h := NewMatchHandler(svc)

		goals1, goals2 := int64(3), int64(2)
		round := int64(1)
		roundIndex := int64(0)
		req := connect.NewRequest(&v1.CreateMatchRequest{
			ContestSlug: "world-cup-2026",
			Match: &v1.Match{
				Country1:      &v1.Country{Code: "BRA", FullName: "Brazil"},
				Country2:      &v1.Country{Code: "ARG", FullName: "Argentina"},
				Country1Goals: &goals1,
				Country2Goals: &goals2,
				Round:         round,
				RoundIndex:    &roundIndex,
			},
		})
		req.Header().Set("Authorization", "Bearer "+superToken)

		resp, err := h.CreateMatch(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "BRA", capturedMatch.Country1.Code)
		assert.Equal(t, "ARG", capturedMatch.Country2.Code)
		assert.Equal(t, 3, *capturedMatch.Country1Goals)
		assert.Equal(t, 2, *capturedMatch.Country2Goals)
		assert.Equal(t, 1, capturedMatch.Round)
		assert.Equal(t, 0, *capturedMatch.RoundIndex)
	})

	t.Run("forbidden for normal user", func(t *testing.T) {
		svc := &mockMatchService{
			createMatchFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				return nil // Should not be called
			},
		}
		h := NewMatchHandler(svc)

		req := connect.NewRequest(&v1.CreateMatchRequest{
			ContestSlug: "world-cup-2026",
			Match:       &v1.Match{},
		})
		req.Header().Set("Authorization", "Bearer "+normalToken)

		_, err := h.CreateMatch(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	})

	t.Run("unauthenticated with no token", func(t *testing.T) {
		svc := &mockMatchService{
			createMatchFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				return nil // Should not be called
			},
		}
		h := NewMatchHandler(svc)

		req := connect.NewRequest(&v1.CreateMatchRequest{
			ContestSlug: "world-cup-2026",
			Match:       &v1.Match{},
		})

		_, err := h.CreateMatch(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("service error propagation", func(t *testing.T) {
		svc := &mockMatchService{
			createMatchFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				return errors.New("update match failed")
			},
		}
		h := NewMatchHandler(svc)

		req := connect.NewRequest(&v1.CreateMatchRequest{
			ContestSlug: "world-cup-2026",
			Match:       &v1.Match{},
		})
		req.Header().Set("Authorization", "Bearer "+superToken)

		_, err := h.CreateMatch(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, "update match failed", err.Error())
	})
}
