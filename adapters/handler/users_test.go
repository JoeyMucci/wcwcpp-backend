package handler

import (
	"context"
	"errors"
	"os"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockUsersService struct {
	ports.UsersService
	countUsersFunc func(ctx context.Context) (int64, error)
	deleteUserFunc func(ctx context.Context, userID string) error
}

func (m *mockUsersService) CountUsers(ctx context.Context) (int64, error) {
	return m.countUsersFunc(ctx)
}

func (m *mockUsersService) DeleteUser(ctx context.Context, userID string) error {
	return m.deleteUserFunc(ctx, userID)
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

func TestUsersHandler_DeleteUser(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")

	userToken := generateTestToken("test_secret", "user1", "normal@example.com")

	tests := []struct {
		name        string
		token       string
		mockFunc    func(ctx context.Context, userID string) error
		expectError bool
	}{
		{
			name:  "success",
			token: userToken,
			mockFunc: func(ctx context.Context, userID string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:  "error",
			token: userToken,
			mockFunc: func(ctx context.Context, userID string) error {
				return errors.New("service error")
			},
			expectError: true,
		},
		{
			name:  "unauthenticated",
			token: "",
			mockFunc: func(ctx context.Context, userID string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockUsersService{deleteUserFunc: tt.mockFunc}
			h := NewUsersHandler(svc)
			req := connect.NewRequest(&v1.DeleteUserRequest{})
			req.Header().Set("Authorization", "Bearer "+tt.token)

			_, err := h.DeleteUser(context.Background(), req)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
