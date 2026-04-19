package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/core/service"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct {
	loginFunc func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error)
}

func (m *mockAuthService) Login(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
	return m.loginFunc(ctx, googleIDToken, username)
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		reqToken     string
		reqUsername  *string
		mockLogin    func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error)
		expectCode   connect.Code
		expectErr    bool
		expectToken  string
		expectUserID string
	}{
		{
			name:     "Success",
			reqToken: "valid_token",
			mockLogin: func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
				return "jwt_token", &entity.User{ID: "123", Email: "a@b.c", Username: "user"}, nil
			},
			expectErr:    false,
			expectToken:  "jwt_token",
			expectUserID: "123",
		},
		{
			name:     "User Not Found",
			reqToken: "valid_token",
			mockLogin: func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
				return "", nil, service.ErrUserNotFound
			},
			expectErr:  true,
			expectCode: connect.CodeNotFound,
		},
		{
			name:     "Invalid Google Token",
			reqToken: "invalid_token",
			mockLogin: func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
				return "", nil, service.ErrInvalidToken
			},
			expectErr:  true,
			expectCode: connect.CodeUnauthenticated,
		},
		{
			name:     "Internal Error",
			reqToken: "valid_token",
			mockLogin: func(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
				return "", nil, errors.New("db error")
			},
			expectErr:  true,
			expectCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{loginFunc: tt.mockLogin}
			h := NewAuthHandler(svc)

			req := connect.NewRequest(&v1.LoginRequest{
				GoogleIdToken: tt.reqToken,
				Username:      tt.reqUsername,
			})

			res, err := h.Login(context.Background(), req)
			if tt.expectErr {
				require.Error(t, err)
				var connectErr *connect.Error
				require.True(t, errors.As(err, &connectErr))
				require.Equal(t, tt.expectCode, connectErr.Code())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectToken, res.Msg.AccessToken)
				require.Equal(t, tt.expectUserID, res.Msg.User.Id)
			}
		})
	}
}
