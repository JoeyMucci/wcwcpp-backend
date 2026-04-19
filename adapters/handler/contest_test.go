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

type mockContestService struct {
	ports.ContestService
	listContestsFunc     func(ctx context.Context) ([]entity.Contest, error)
	listSubcontestsFunc  func(ctx context.Context, contestSlug string) ([]entity.Contest, error)
	createSubcontestFunc func(ctx context.Context, contestSlug string, title string) (string, error)
	deleteSubcontestFunc func(ctx context.Context, subcontestSlug string) error
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
				return nil, errors.New("service error")
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
