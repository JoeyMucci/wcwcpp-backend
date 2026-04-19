package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockUsersService struct {
	ports.UsersService
	countUsersFunc func(ctx context.Context) (int64, error)
}

func (m *mockUsersService) CountUsers(ctx context.Context) (int64, error) {
	return m.countUsersFunc(ctx)
}

func TestUsersHandler_CountUsers(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context) (int64, error)
		expectCount int64
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) (int64, error) {
				return 42, nil
			},
			expectCount: 42,
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context) (int64, error) {
				return 0, errors.New("service error")
			},
			expectCount: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockUsersService{countUsersFunc: tt.mockFunc}
			h := NewUsersHandler(svc)
			
			resp, err := h.CountUsers(context.Background(), connect.NewRequest(&v1.CountUsersRequest{}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if !tt.expectError && resp.Msg.Count != tt.expectCount {
				t.Errorf("expected count %d, got %d", tt.expectCount, resp.Msg.Count)
			}
		})
	}
}
