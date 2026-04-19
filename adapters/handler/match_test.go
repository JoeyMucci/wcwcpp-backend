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
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
				return []entity.Match{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, letter string) ([]entity.Match, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockMatchService{listGroupMatchesFunc: tt.mockFunc}
			h := NewMatchHandler(svc)
			
			_, err := h.ListGroupMatches(context.Background(), connect.NewRequest(&v1.ListGroupMatchesRequest{ContestSlug: "test", Letter: "A"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestMatchHandler_ListKnockoutMatches(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string) ([]entity.Match, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.Match, error) {
				return []entity.Match{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.Match, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockMatchService{listKnockoutMatchesFunc: tt.mockFunc}
			h := NewMatchHandler(svc)
			
			_, err := h.ListKnockoutMatches(context.Background(), connect.NewRequest(&v1.ListKnockoutMatchesRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestMatchHandler_CreateMatch(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, match entity.Match) error
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, match entity.Match) error {
				return errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockMatchService{createMatchFunc: tt.mockFunc}
			h := NewMatchHandler(svc)
			
			_, err := h.CreateMatch(context.Background(), connect.NewRequest(&v1.CreateMatchRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
