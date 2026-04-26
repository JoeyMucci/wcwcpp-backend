package handler

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

func generateTestToken(secret, userID, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

type mockContestService struct {
	ports.ContestService
	listContestsFunc     func(ctx context.Context) ([]entity.Contest, error)
	createContestFunc    func(ctx context.Context, contest entity.Contest) error
	listSubcontestsFunc  func(ctx context.Context, contestSlug string) ([]entity.Contest, error)
	createSubcontestFunc func(ctx context.Context, contestSlug string, title string) (string, error)
	deleteSubcontestFunc func(ctx context.Context, subcontestSlug string) error
}

func (m *mockContestService) CreateContest(ctx context.Context, contest entity.Contest) error {
	return m.createContestFunc(ctx, contest)
}

func (m *mockContestService) ListContests(ctx context.Context) ([]entity.Contest, error) {
	return m.listContestsFunc(ctx)
}
func (m *mockContestService) ListSubcontests(ctx context.Context, contestSlug string) ([]entity.Contest, error) {
	return m.listSubcontestsFunc(ctx, contestSlug)
}
func (m *mockContestService) CreateSubcontest(ctx context.Context, contestSlug string, title string) (string, error) {
	return m.createSubcontestFunc(ctx, contestSlug, title)
}
func (m *mockContestService) DeleteSubcontest(ctx context.Context, subcontestSlug string) error {
	return m.deleteSubcontestFunc(ctx, subcontestSlug)
}

func TestContestHandler_ListContests(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context) ([]entity.Contest, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) ([]entity.Contest, error) {
				return []entity.Contest{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context) ([]entity.Contest, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContestService{listContestsFunc: tt.mockFunc}
			h := NewContestHandler(svc)

			_, err := h.ListContests(context.Background(), connect.NewRequest(&v1.ListContestsRequest{}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestContestHandler_CreateContest(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	os.Setenv("SUPERADMIN_EMAILS", "super1@example.com")

	superToken := generateTestToken("test_secret", "user1", "super1@example.com")
	normalToken := generateTestToken("test_secret", "user2", "normal@example.com")

	tests := []struct {
		name        string
		token       string
		mockFunc    func(ctx context.Context, contest entity.Contest) error
		expectError bool
		errCode     connect.Code
	}{
		{
			name:  "success superadmin",
			token: superToken,
			mockFunc: func(ctx context.Context, contest entity.Contest) error {
				return nil
			},
			expectError: false,
		},
		{
			name:  "forbidden normal user",
			token: normalToken,
			mockFunc: func(ctx context.Context, contest entity.Contest) error {
				return nil // Should not be called
			},
			expectError: true,
			errCode:     connect.CodePermissionDenied,
		},
		{
			name:  "error",
			token: superToken,
			mockFunc: func(ctx context.Context, contest entity.Contest) error {
				return errors.New("error")
			},
			expectError: true,
			errCode:     connect.CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContestService{createContestFunc: tt.mockFunc}
			h := NewContestHandler(svc)

			req := connect.NewRequest(&v1.CreateContestRequest{
				Title: "Test Contest",
				Groups: []*v1.Group{
					{
						Letter: "A",
						Countries: []*v1.Country{
							{Code: "USA", FullName: "United States"},
						},
					},
				},
			})
			req.Header().Set("Authorization", "Bearer "+tt.token)

			_, err := h.CreateContest(context.Background(), req)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if tt.expectError && tt.errCode != 0 && connect.CodeOf(err) != tt.errCode {
				t.Errorf("expected error code %v, got %v", tt.errCode, connect.CodeOf(err))
			}
		})
	}
}

func TestContestHandler_ListSubcontests(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string) ([]entity.Contest, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.Contest, error) {
				return []entity.Contest{{}}, nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string) ([]entity.Contest, error) {
				return nil, errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContestService{listSubcontestsFunc: tt.mockFunc}
			h := NewContestHandler(svc)

			_, err := h.ListSubcontests(context.Background(), connect.NewRequest(&v1.ListSubcontestsRequest{ContestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestContestHandler_CreateSubcontest(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, contestSlug string, title string) (string, error)
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, contestSlug string, title string) (string, error) {
				return "JOINCODE", nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, contestSlug string, title string) (string, error) {
				return "", errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContestService{createSubcontestFunc: tt.mockFunc}
			h := NewContestHandler(svc)

			resp, err := h.CreateSubcontest(context.Background(), connect.NewRequest(&v1.CreateSubcontestRequest{ContestSlug: "test", SubcontestTitle: "title"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if !tt.expectError && resp.Msg.JoinCode != "JOINCODE" {
				t.Errorf("expected join code JOINCODE, got %s", resp.Msg.JoinCode)
			}
		})
	}
}

func TestContestHandler_DeleteSubcontest(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, subcontestSlug string) error
		expectError bool
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context, subcontestSlug string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "error",
			mockFunc: func(ctx context.Context, subcontestSlug string) error {
				return errors.New("service error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContestService{deleteSubcontestFunc: tt.mockFunc}
			h := NewContestHandler(svc)

			_, err := h.DeleteSubcontest(context.Background(), connect.NewRequest(&v1.DeleteSubcontestRequest{SubcontestSlug: "test"}))
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
