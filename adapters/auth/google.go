package auth

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/idtoken"
)

type GoogleTokenValidator struct {
	clientID string
}

func NewGoogleTokenValidator() *GoogleTokenValidator {
	return &GoogleTokenValidator{
		clientID: os.Getenv("GOOGLE_CLIENT_ID"),
	}
}

func (v *GoogleTokenValidator) ValidateGoogleToken(ctx context.Context, token string) (string, error) {
	payload, err := idtoken.Validate(ctx, token, v.clientID)
	if err != nil {
		return "", fmt.Errorf("invalid google id token: %w", err)
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return "", fmt.Errorf("email not found in google token claims")
	}

	return email, nil
}
