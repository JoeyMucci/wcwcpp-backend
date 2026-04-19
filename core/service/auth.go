package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
)

var (
	ErrInvalidToken = errors.New("invalid google token")
	ErrUserNotFound = errors.New("user not found, username required for registration")
)

type AuthService struct {
	repo           ports.UserRepository
	tokenValidator ports.TokenValidator
	jwtSecret      []byte
}

func NewAuthService(repo ports.UserRepository, tokenValidator ports.TokenValidator) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback_secret_for_dev_only"
	}
	return &AuthService{
		repo:           repo,
		tokenValidator: tokenValidator,
		jwtSecret:      []byte(secret),
	}
}

func (s *AuthService) Login(ctx context.Context, googleIDToken string, username *string) (string, *entity.User, error) {
	email, err := s.tokenValidator.ValidateGoogleToken(ctx, googleIDToken)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if user == nil {
		if username == nil || *username == "" {
			return "", nil, ErrUserNotFound
		}

		user, err = s.repo.CreateUser(ctx, email, *username)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(24 * 7 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	return tokenString, user, nil
}
