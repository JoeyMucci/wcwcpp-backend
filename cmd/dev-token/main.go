package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback_secret_for_dev_only"
	}

	// Default to a test email, or use the first argument
	email := "superadmin@example.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	// Create a token with a 1-year expiration for dev convenience
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   uuid.New().String(),
		"email": email,
		"exp":   time.Now().Add(24 * 365 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Dev JWT generated for email: %s\n", email)
	fmt.Printf("If testing superadmin, ensure this email is in your SUPERADMIN_EMAILS env var.\n\n")
	fmt.Printf("Header to use in Postman/curl/frontend:\n")
	fmt.Printf("Authorization: Bearer %s\n", tokenString)
}
