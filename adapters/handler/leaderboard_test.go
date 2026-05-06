package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockLeaderboardService struct {
	ports.LeaderboardService
	leaderboardFunc    func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
	subleaderboardFunc func(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
}

func (m *mockLeaderboardService) Leaderboard(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.leaderboardFunc(ctx, contestSlug, limit, offset)
}

func (m *mockLeaderboardService) Subleaderboard(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
	return m.subleaderboardFunc(ctx, subcontestSlug, limit, offset)
}

func TestLeaderboardHandler_Leaderboard(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return map[string][]entity.LeaderboardEntry{"group": []entity.LeaderboardEntry{{}}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
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
			if !tt.expectError && len(resp.Msg.Group) == 0 {
				t.Errorf("expected group to be non-empty")
			}
		})
	}
}

func TestLeaderboardHandler_Subleaderboard(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return map[string][]entity.LeaderboardEntry{"group": []entity.LeaderboardEntry{{}}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, subcontestSlug string, limit int32, offset int32) (map[string][]entity.LeaderboardEntry, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLeaderboardService{subleaderboardFunc: tt.mockFunc}
			h := NewLeaderboardHandler(svc)

			resp, err := h.Subleaderboard(context.Background(), connect.NewRequest(&v1.SubleaderboardRequest{SubcontestSlug: "test", Limit: 10, Offset: 0}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if !tt.expectError && len(resp.Msg.Group) == 0 {
				t.Errorf("expected group to be non-empty")
			}
		})
	}
}
