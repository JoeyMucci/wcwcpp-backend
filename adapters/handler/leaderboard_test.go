package handler

import (
	"context"
	"errors"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockLeaderboardService struct {
	ports.LeaderboardService
	leaderboardFunc    func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	subleaderboardFunc func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
}

func (m *mockLeaderboardService) Leaderboard(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.leaderboardFunc(ctx, contestSlug, limit, offset)
}

func (m *mockLeaderboardService) Subleaderboard(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.subleaderboardFunc(ctx, userID, subcontestSlug, limit, offset)
}

func TestLeaderboardHandler_Leaderboard(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
		expectError bool
		errCode     connect.Code
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return map[string][]entity.LeaderboardEntry{"group": {{}}}, nil
			},
			expectError: false,
		},
		{
			name: "contest not found",
			mockFunc: func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("contest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
			errCode:     connect.CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLeaderboardService{leaderboardFunc: tt.mockFunc}
			h := NewLeaderboardHandler(svc)

			resp, err := h.Leaderboard(context.Background(), connect.NewRequest(&v1.LeaderboardRequest{ContestSlug: "test", Limit: 10, Offset: 0}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if tt.expectError && tt.errCode != 0 && connect.CodeOf(err) != tt.errCode {
				t.Errorf("expected error code %v, got %v", tt.errCode, connect.CodeOf(err))
			}
			if !tt.expectError && len(resp.Msg.Group) == 0 {
				t.Errorf("expected group to be non-empty")
			}
		})
	}
}

func TestLeaderboardHandler_Subleaderboard(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	validToken := generateTestToken("test_secret", "user-123", "normal@example.com")

	tests := []struct {
		name        string
		token       string
		mockFunc    func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
		expectError bool
		errCode     connect.Code
	}{
		{
			name:  "success",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				if userID != "user-123" {
					return nil, errors.New("unexpected userID in mock")
				}
				return map[string][]entity.LeaderboardEntry{"group": {{}}}, nil
			},
			expectError: false,
		},
		{
			name:  "permission denied",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("permission denied: no access to subcontest")
			},
			expectError: true,
			errCode:     connect.CodePermissionDenied,
		},
		{
			name:  "subcontest not found",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("subcontest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
		{
			name:  "service error",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
			errCode:     connect.CodeUnknown,
		},
		{
			name:  "unauthenticated missing token",
			token: "",
			mockFunc: func(ctx context.Context, userID string, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, nil
			},
			expectError: true,
			errCode:     connect.CodeUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLeaderboardService{subleaderboardFunc: tt.mockFunc}
			h := NewLeaderboardHandler(svc)

			req := connect.NewRequest(&v1.SubleaderboardRequest{SubcontestSlug: "test", Limit: 10, Offset: 0})
			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			resp, err := h.Subleaderboard(context.Background(), req)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error %v, got %v", tt.expectError, err)
			}
			if tt.expectError && tt.errCode != 0 && connect.CodeOf(err) != tt.errCode {
				t.Errorf("expected error code %v, got %v", tt.errCode, connect.CodeOf(err))
			}
			if !tt.expectError && len(resp.Msg.Group) == 0 {
				t.Errorf("expected group to be non-empty")
			}
		})
	}
}
