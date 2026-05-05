package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	secret := os.Getenv("JWT_SECRET")

	email := "superadmin@example.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stderr, "DATABASE_URL environment variable is required\n")
		os.Exit(1)
	}
	
	if secret == "" {
		fmt.Fprintf(os.Stderr, "JWT_SECRET environment variable is required\n")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var userID string
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err == sql.ErrNoRows {
		username := strings.Split(email, "@")[0]
		err = db.QueryRow("INSERT INTO users (email, username) VALUES ($1, $2) RETURNING id", email, username).Scan(&userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying user: %v\n", err)
		os.Exit(1)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(24 * 365 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(tokenString)
}
