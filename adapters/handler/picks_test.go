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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPicksService struct {
	ports.PicksService
	listGroupPicksFunc      func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error)
	createGroupPicksFunc    func(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error
	listKnockoutPicksFunc   func(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error)
	createKnockoutPicksFunc func(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error
}

func (m *mockPicksService) ListGroupPicks(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
	return m.listGroupPicksFunc(ctx, userID, contestSlug)
}

func (m *mockPicksService) CreateGroupPicks(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error {
	return m.createGroupPicksFunc(ctx, userID, contestSlug, picks)
}

func (m *mockPicksService) ListKnockoutPicks(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error) {
	return m.listKnockoutPicksFunc(ctx, userID, contestSlug)
}

func (m *mockPicksService) CreateKnockoutPicks(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error {
	return m.createKnockoutPicksFunc(ctx, userID, contestSlug, pick)
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
				require.Equal(t, "user-123", userID)
				require.Equal(t, "world-cup-2026", contestSlug)
				return samplePicks, sampleStandings, nil
			},
			expectError: false,
			assertResp: func(t *testing.T, resp *connect.Response[v1.ListGroupPicksResponse]) {
				require.Len(t, resp.Msg.Picks, 1)
				pick := resp.Msg.Picks[0]
				assert.Equal(t, "A", pick.Group.Letter)
				require.Len(t, pick.Group.Countries, 4)
				assert.Equal(t, "USA", pick.Group.Countries[0].Code)
				assert.True(t, pick.ExtraQualifier)

				require.Len(t, resp.Msg.Results, 1)
				rg := resp.Msg.Results[0]
				assert.Equal(t, "A", rg.Letter)
				require.Len(t, rg.RankedCountries, 4)
				assert.Equal(t, "USA", rg.RankedCountries[0].Code)
				assert.Equal(t, int64(9), rg.RankedCountries[0].Points)
			},
		},
		{
			name:  "success — finalized group standings and wildcard",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) ([]entity.GroupPick, []entity.GroupStanding, error) {
				rank1 := int32(1)
				rank2 := int32(2)
				rank3 := int32(3)
				rank4 := int32(4)
				isQual := true

				standings := []entity.GroupStanding{
					{Country: entity.Country{Code: "USA", FullName: "United States"}, Letter: "A", Points: 9, Wins: 3, Rank: &rank1},
					{Country: entity.Country{Code: "MEX", FullName: "Mexico"}, Letter: "A", Points: 6, Wins: 2, Rank: &rank2},
					{Country: entity.Country{Code: "CAN", FullName: "Canada"}, Letter: "A", Points: 3, Wins: 1, Rank: &rank3, IsThirdPlaceQualifier: &isQual},
					{Country: entity.Country{Code: "ARG", FullName: "Argentina"}, Letter: "A", Points: 0, Rank: &rank4},
				}
				return samplePicks, standings, nil
			},
			expectError: false,
			assertResp: func(t *testing.T, resp *connect.Response[v1.ListGroupPicksResponse]) {
				require.Len(t, resp.Msg.Results, 1)
				rg := resp.Msg.Results[0]
				assert.Equal(t, "A", rg.Letter)
				assert.True(t, rg.Finalized)
				assert.True(t, rg.ExtraQualifierFinalized)
				require.NotNil(t, rg.ExtraQualifier)
				assert.True(t, *rg.ExtraQualifier)
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
				require.Error(t, err)
				if tt.errCode != 0 {
					assert.Equal(t, tt.errCode, connect.CodeOf(err))
				}
			} else {
				require.NoError(t, err)
				if tt.assertResp != nil {
					tt.assertResp(t, resp)
				}
			}
		})
	}
}

func TestPicksHandler_CreateGroupPicks(t *testing.T) {
	validToken := generateTestToken("secret", "user-123", "user@example.com")

	tests := []struct {
		name        string
		token       string
		reqPayload  *v1.CreateGroupPicksRequest
		mockFunc    func(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error
		expectError bool
		errCode     connect.Code
	}{
		{
			name:  "success",
			token: validToken,
			reqPayload: &v1.CreateGroupPicksRequest{
				ContestSlug: "world-cup-2026",
				Picks: []*v1.GroupPick{
					{
						Group: &v1.Group{
							Letter: "A",
							Countries: []*v1.Country{
								{Code: "USA", FullName: "United States"},
								{Code: "MEX", FullName: "Mexico"},
							},
						},
						ExtraQualifier: true,
					},
				},
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error {
				require.Equal(t, "user-123", userID)
				require.Equal(t, "world-cup-2026", contestSlug)
				require.Len(t, picks, 1)
				assert.Equal(t, "A", picks[0].Letter)
				assert.True(t, picks[0].ExtraQualifier)
				require.Len(t, picks[0].Entries, 2)
				assert.Equal(t, "USA", picks[0].Entries[0].Country.Code)
				assert.Equal(t, 1, picks[0].Entries[0].Place)
				return nil
			},
			expectError: false,
		},
		{
			name:  "unauthenticated",
			token: "",
			reqPayload: &v1.CreateGroupPicksRequest{
				ContestSlug: "world-cup-2026",
			},
			expectError: true,
			errCode:     connect.CodeUnauthenticated,
		},
		{
			name:  "contest not found",
			token: validToken,
			reqPayload: &v1.CreateGroupPicksRequest{
				ContestSlug: "nonexistent",
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error {
				return errors.New("contest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
		{
			name:  "picks locked",
			token: validToken,
			reqPayload: &v1.CreateGroupPicksRequest{
				ContestSlug: "world-cup-2026",
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, picks []entity.GroupPick) error {
				return errors.New("group stage picks are locked")
			},
			expectError: true,
			errCode:     connect.CodeFailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "secret")

			svc := &mockPicksService{createGroupPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)

			req := connect.NewRequest(tt.reqPayload)
			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			handler := interceptor.WithAuth(func(ctx context.Context, r *connect.Request[v1.CreateGroupPicksRequest]) (*connect.Response[v1.CreateGroupPicksResponse], error) {
				return h.CreateGroupPicks(ctx, r)
			})

			_, err := handler(context.Background(), req)
			if tt.expectError {
				require.Error(t, err)
				if tt.errCode != 0 {
					assert.Equal(t, tt.errCode, connect.CodeOf(err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPicksHandler_ListKnockoutPicks(t *testing.T) {
	validToken := generateTestToken("secret", "user-123", "user@example.com")

	samplePick := entity.KnockoutPick{
		Entries: []entity.KnockoutPickEntry{
			{Country: entity.Country{Code: "USA"}, Round: 16},
		},
	}
	sampleResult := entity.KnockoutPick{
		Entries: []entity.KnockoutPickEntry{
			{Country: entity.Country{Code: "MEX"}, Round: 8},
		},
	}

	tests := []struct {
		name        string
		token       string
		mockFunc    func(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error)
		expectError bool
		errCode     connect.Code
		assertResp  func(t *testing.T, resp *connect.Response[v1.ListKnockoutPicksResponse])
	}{
		{
			name:  "success",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error) {
				require.Equal(t, "user-123", userID)
				require.Equal(t, "world-cup-2026", contestSlug)
				return samplePick, sampleResult, nil
			},
			expectError: false,
			assertResp: func(t *testing.T, resp *connect.Response[v1.ListKnockoutPicksResponse]) {
				require.NotNil(t, resp.Msg.Pick)
				require.Len(t, resp.Msg.Pick.Entries, 1)
				assert.Equal(t, "USA", resp.Msg.Pick.Entries[0].Country.Code)
				assert.Equal(t, int64(16), resp.Msg.Pick.Entries[0].Round)

				require.NotNil(t, resp.Msg.Result)
				require.Len(t, resp.Msg.Result.Entries, 1)
				assert.Equal(t, "MEX", resp.Msg.Result.Entries[0].Country.Code)
				assert.Equal(t, int64(8), resp.Msg.Result.Entries[0].Round)
			},
		},
		{
			name:        "unauthenticated",
			token:       "",
			expectError: true,
			errCode:     connect.CodeUnauthenticated,
		},
		{
			name:  "contest not found",
			token: validToken,
			mockFunc: func(ctx context.Context, userID string, contestSlug string) (entity.KnockoutPick, entity.KnockoutPick, error) {
				return entity.KnockoutPick{}, entity.KnockoutPick{}, errors.New("contest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "secret")

			svc := &mockPicksService{listKnockoutPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)

			req := connect.NewRequest(&v1.ListKnockoutPicksRequest{ContestSlug: "world-cup-2026"})
			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			handler := interceptor.WithAuth(func(ctx context.Context, r *connect.Request[v1.ListKnockoutPicksRequest]) (*connect.Response[v1.ListKnockoutPicksResponse], error) {
				return h.ListKnockoutPicks(ctx, r)
			})

			resp, err := handler(context.Background(), req)
			if tt.expectError {
				require.Error(t, err)
				if tt.errCode != 0 {
					assert.Equal(t, tt.errCode, connect.CodeOf(err))
				}
			} else {
				require.NoError(t, err)
				if tt.assertResp != nil {
					tt.assertResp(t, resp)
				}
			}
		})
	}
}

func TestPicksHandler_CreateKnockoutPicks(t *testing.T) {
	validToken := generateTestToken("secret", "user-123", "user@example.com")

	tests := []struct {
		name        string
		token       string
		reqPayload  *v1.CreateKnockoutPicksRequest
		mockFunc    func(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error
		expectError bool
		errCode     connect.Code
	}{
		{
			name:  "success",
			token: validToken,
			reqPayload: &v1.CreateKnockoutPicksRequest{
				ContestSlug: "world-cup-2026",
				Pick: &v1.KnockoutPick{
					Entries: []*v1.KnockoutEntry{
						{
							Country: &v1.Country{Code: "USA"},
							Round:   16,
						},
					},
				},
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error {
				require.Equal(t, "user-123", userID)
				require.Equal(t, "world-cup-2026", contestSlug)
				require.Len(t, pick.Entries, 1)
				assert.Equal(t, "USA", pick.Entries[0].Country.Code)
				assert.Equal(t, 16, pick.Entries[0].Round)
				return nil
			},
			expectError: false,
		},
		{
			name:  "contest not found",
			token: validToken,
			reqPayload: &v1.CreateKnockoutPicksRequest{
				ContestSlug: "nonexistent",
				Pick: &v1.KnockoutPick{
					Entries: []*v1.KnockoutEntry{
						{
							Country: &v1.Country{Code: "USA"},
							Round:   16,
						},
					},
				},
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error {
				return errors.New("contest not found")
			},
			expectError: true,
			errCode:     connect.CodeNotFound,
		},
		{
			name:  "picks locked",
			token: validToken,
			reqPayload: &v1.CreateKnockoutPicksRequest{
				ContestSlug: "world-cup-2026",
				Pick: &v1.KnockoutPick{
					Entries: []*v1.KnockoutEntry{
						{
							Country: &v1.Country{Code: "USA"},
							Round:   16,
						},
					},
				},
			},
			mockFunc: func(ctx context.Context, userID string, contestSlug string, pick entity.KnockoutPick) error {
				return errors.New("knockout stage picks are locked")
			},
			expectError: true,
			errCode:     connect.CodeFailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "secret")

			svc := &mockPicksService{createKnockoutPicksFunc: tt.mockFunc}
			h := NewPicksHandler(svc)

			req := connect.NewRequest(tt.reqPayload)
			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			handler := interceptor.WithAuth(func(ctx context.Context, r *connect.Request[v1.CreateKnockoutPicksRequest]) (*connect.Response[v1.CreateKnockoutPicksResponse], error) {
				return h.CreateKnockoutPicks(ctx, r)
			})

			_, err := handler(context.Background(), req)
			if tt.expectError {
				require.Error(t, err)
				if tt.errCode != 0 {
					assert.Equal(t, tt.errCode, connect.CodeOf(err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
