package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockLeaderboardService struct {
	ports.LeaderboardService
	leaderboardFunc    func(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
	subleaderboardFunc func(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
}

func (m *mockLeaderboardService) Leaderboard(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
	return m.leaderboardFunc(ctx, contestSlug, pageSize, pageToken)
}

func (m *mockLeaderboardService) Subleaderboard(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
	return m.subleaderboardFunc(ctx, subcontestSlug, pageSize, pageToken)
}

func TestLeaderboardHandler_Leaderboard(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
				return []entity.LeaderboardEntry{{}}, "next_token", nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
				return nil, "", errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLeaderboardService{leaderboardFunc: tt.mockFunc}
			h := NewLeaderboardHandler(svc)
			
			resp, err := h.Leaderboard(context.Background(), connect.NewRequest(&v1.LeaderboardRequest{ContestSlug: "test", PageSize: 10, PageToken: "token"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if !tt.expectError && resp.Msg.NextPageToken != "next_token" {
				t.Errorf("expected token next_token, got %s", resp.Msg.NextPageToken)
			}
		})
	}
}

func TestLeaderboardHandler_Subleaderboard(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
				return []entity.LeaderboardEntry{{}}, "next_token", nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, subcontestSlug string, pageSize int32, pageToken string) ([]entity.LeaderboardEntry, string, error) {
				return nil, "", errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLeaderboardService{subleaderboardFunc: tt.mockFunc}
			h := NewLeaderboardHandler(svc)
			
			resp, err := h.Subleaderboard(context.Background(), connect.NewRequest(&v1.SubleaderboardRequest{SubcontestSlug: "test", PageSize: 10, PageToken: "token"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if !tt.expectError && resp.Msg.NextPageToken != "next_token" {
				t.Errorf("expected token next_token, got %s", resp.Msg.NextPageToken)
			}
		})
	}
}
