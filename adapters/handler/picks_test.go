package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/interceptor"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/ports"
)

type mockPicksService struct {
	ports.PicksService
	listGroupPicksFunc func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error)
}

func (m *mockPicksService) ListGroupPicks(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
	return m.listGroupPicksFunc(ctx, userID, contestSlug)
}

func TestPicksHandler_ListGroupPicks(t *testing.T) {
	validToken := generateTestToken("secret", "user-123", "user@example.com")

	samplePicks := []entity.GroupPick{
		{
			Letter: "A",
			Entries: []entity.GroupPickEntry{
				{Country: entity.Country{Code: "USA", FullName: "United States"}, Place: 1},
				{Country: entity.Country{Code: "MEX", FullName: "Mexico"}, Place: 2},
				{Country: entity.Country{Code: "CAN", FullName: "Canada"}, Place: 3},
				{Country: entity.Country{Code: "ARG", FullName: "Argentina"}, Place: 4},
			},
			ExtraQualifier: true,
		},
	}
	sampleStandings := []entity.GroupStanding{
		{Country: entity.Country{Code: "USA", FullName: "United States"}, Letter: "A", Points: 9, Wins: 3},
		{Country: entity.Country{Code: "MEX", FullName: "Mexico"}, Letter: "A", Points: 6, Wins: 2},
		{Country: entity.Country{Code: "CAN", FullName: "Canada"}, Letter: "A", Points: 3, Wins: 1},
		{Country: entity.Country{Code: "ARG", FullName: "Argentina"}, Letter: "A", Points: 0},
	}

	tests := []struct {
		name        string
		token       string
		mockFunc    func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error)
		expectError bool
		errCode     connect.Code
		assertResp  func(t *testing.T, resp *connect.Response[v1.ListGroupPicksResponse])
	}{
		{
			name:  "success — picks and standings mapped correctly",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
				if userID != "user-123" {
					t.Errorf("expected userID 'user-123', got %q", userID)
				}
				if contestSlug != "world-cup-2026" {
					t.Errorf("expected slug 'world-cup-2026', got %q", contestSlug)
				}
				return samplePicks, sampleStandings, nil
			},
			expectError: false,
			assertResp: func(t *testing.T, resp *connect.Response[v1.ListGroupPicksResponse]) {
				// Picks
				if len(resp.Msg.Picks) != 1 {
					t.Fatalf("expected 1 pick group, got %d", len(resp.Msg.Picks))
				}
				pick := resp.Msg.Picks[0]
				if pick.Group.Letter != "A" {
					t.Errorf("expected letter A, got %s", pick.Group.Letter)
				}
				if len(pick.Group.Countries) != 4 {
					t.Errorf("expected 4 countries, got %d", len(pick.Group.Countries))
				}
				if pick.Group.Countries[0].Code != "USA" {
					t.Errorf("expected first country USA, got %s", pick.Group.Countries[0].Code)
				}
				if !pick.ExtraQualifier {
					t.Error("expected extra_qualifier=true")
				}

				// Standings
				if len(resp.Msg.RankedGroups) != 1 {
					t.Fatalf("expected 1 ranked group, got %d", len(resp.Msg.RankedGroups))
				}
				rg := resp.Msg.RankedGroups[0]
				if rg.Letter != "A" {
					t.Errorf("expected letter A, got %s", rg.Letter)
				}
				if len(rg.RankedCountries) != 4 {
					t.Errorf("expected 4 ranked countries, got %d", len(rg.RankedCountries))
				}
				if rg.RankedCountries[0].Code != "USA" {
					t.Errorf("expected first ranked country USA, got %s", rg.RankedCountries[0].Code)
				}
				if rg.RankedCountries[0].Points != 9 {
					t.Errorf("expected points 9, got %d", rg.RankedCountries[0].Points)
				}
				if rg.RankedCountries[0].Wins != 3 {
					t.Errorf("expected wins 3, got %d", rg.RankedCountries[0].Wins)
				}
			},
		},
		{
			name:  "contest not found",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
				return nil, nil, errors.New("contest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
		{
			name:  "service error",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
				return nil, nil, errors.New("unexpected db failure")
			},
			expectError: true,
			errCode:     connect.CodeUnknown,
		},
		{
			name:        "unauthenticated",
			token:       "",
			mockFunc:    nil,
			expectError: true,
			errCode:     connect.CodeUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "secret")

			svc := &mockPicksService{listGroupPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)

			req := connect.NewRequest(&v1.ListGroupPicksRequest{ContestSlug: "world-cup-2026"})
			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			handler := interceptor.WithAuth(func(ctx context.Context, r *connect.Request[v1.ListGroupPicksRequest]) (*connect.Response[v1.ListGroupPicksResponse], error) {
				return h.ListGroupPicks(ctx, r)
			})

			resp, err := handler(context.Background(), req)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCode != 0 && connect.CodeOf(err) != tt.errCode {
					t.Errorf("expected code %v, got %v", tt.errCode, connect.CodeOf(err))
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if tt.assertResp != nil {
					tt.assertResp(t, resp)
				}
			}
		})
	}
}
