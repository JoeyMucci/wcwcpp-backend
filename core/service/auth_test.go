package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/stretchr/testify/require"
)

type mockUserRepository struct {
	findByEmailFunc func(ctx context.Context, email string) (*entity.User, error)
	createUserFunc  func(ctx context.Context, email string, username string) (*entity.User, error)
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return m.findByEmailFunc(ctx, email)
}
func (m *mockUserRepository) CreateUser(ctx context.Context, email string, username string) (*entity.User, error) {
	return m.createUserFunc(ctx, email, username)
}

type mockTokenValidator struct {
	validateFunc func(ctx context.Context, token string) (string, error)
}

func (m *mockTokenValidator) ValidateGoogleToken(ctx context.Context, token string) (string, error) {
	return m.validateFunc(ctx, token)
}

func TestAuthService_Login(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")

	tests := []struct {
		name         string
		token        string
		username     *string
		mockValidate func(ctx context.Context, token string) (string, error)
		mockFind     func(ctx context.Context, email string) (*entity.User, error)
		mockCreate   func(ctx context.Context, email string, username string) (*entity.User, error)
		expectErr    error
		expectUser   *entity.User
		expectToken  bool
	}{
		{
			name:  "User exists, successful login",
			token: "valid_google_token",
			mockValidate: func(ctx context.Context, token string) (string, error) {
				return "test@example.com", nil
			},
			mockFind: func(ctx context.Context, email string) (*entity.User, error) {
				return &entity.User{ID: "1", Email: email, Username: "testuser"}, nil
			},
			expectErr:  nil,
			expectUser: &entity.User{ID: "1", Email: "test@example.com", Username: "testuser"},
		},
		{
			name:  "Invalid google token",
			token: "invalid_google_token",
			mockValidate: func(ctx context.Context, token string) (string, error) {
				return "", errors.New("invalid")
			},
			expectErr: ErrInvalidToken,
		},
		{
			name:  "User not found, no username provided",
			token: "valid_google_token",
			mockValidate: func(ctx context.Context, token string) (string, error) {
				return "new@example.com", nil
			},
			mockFind: func(ctx context.Context, email string) (*entity.User, error) {
				return nil, nil
			},
			expectErr: ErrUserNotFound,
		},
		{
			name:  "User not found, username provided -> created",
			token: "valid_google_token",
			username: func() *string {
				s := "newuser"
				return &s
			}(),
			mockValidate: func(ctx context.Context, token string) (string, error) {
				return "new@example.com", nil
			},
			mockFind: func(ctx context.Context, email string) (*entity.User, error) {
				return nil, nil
			},
			mockCreate: func(ctx context.Context, email string, username string) (*entity.User, error) {
				return &entity.User{ID: "2", Email: email, Username: username}, nil
			},
			expectErr:  nil,
			expectUser: &entity.User{ID: "2", Email: "new@example.com", Username: "newuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{
				findByEmailFunc: tt.mockFind,
				createUserFunc:  tt.mockCreate,
			}
			val := &mockTokenValidator{
				validateFunc: tt.mockValidate,
			}

			svc := NewAuthService(repo, val)

			token, user, err := svc.Login(context.Background(), tt.token, tt.username)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectUser, user)
				require.NotEmpty(t, token)
			}
		})
	}
}
