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

type mockPicksService struct {
	ports.PicksService
	listGroupPicksFunc      func(ctx context.Context, contestSlug string) ([]entity.GroupPick, error)
	createGroupPicksFunc    func(ctx context.Context, contestSlug string, pick entity.GroupPick) error
	listKnockoutPicksFunc   func(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error)
	createKnockoutPicksFunc func(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error
}

func (m *mockPicksService) ListGroupPicks(ctx context.Context, contestSlug string) ([]entity.GroupPick, error) {
	return m.listGroupPicksFunc(ctx, contestSlug)
}
func (m *mockPicksService) CreateGroupPicks(ctx context.Context, contestSlug string, pick entity.GroupPick) error {
	return m.createGroupPicksFunc(ctx, contestSlug, pick)
}
func (m *mockPicksService) ListKnockoutPicks(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error) {
	return m.listKnockoutPicksFunc(ctx, contestSlug)
}
func (m *mockPicksService) CreateKnockoutPicks(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error {
	return m.createKnockoutPicksFunc(ctx, contestSlug, pick)
}

func TestPicksHandler_ListGroupPicks(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string) ([]entity.GroupPick, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.GroupPick, error) {
				return []entity.GroupPick{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.GroupPick, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPicksService{listGroupPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)
			
			_, err := h.ListGroupPicks(context.Background(), connect.NewRequest(&v1.ListGroupPicksRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestPicksHandler_CreateGroupPicks(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, pick entity.GroupPick) error
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, pick entity.GroupPick) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, pick entity.GroupPick) error {
				return errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPicksService{createGroupPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)
			
			_, err := h.CreateGroupPicks(context.Background(), connect.NewRequest(&v1.CreateGroupPicksRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestPicksHandler_ListKnockoutPicks(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error) {
				return []entity.KnockoutPick{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.KnockoutPick, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPicksService{listKnockoutPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)
			
			_, err := h.ListKnockoutPicks(context.Background(), connect.NewRequest(&v1.ListKnockoutPicksRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestPicksHandler_CreateKnockoutPicks(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, pick entity.KnockoutPick) error {
				return errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPicksService{createKnockoutPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)
			
			_, err := h.CreateKnockoutPicks(context.Background(), connect.NewRequest(&v1.CreateKnockoutPicksRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
