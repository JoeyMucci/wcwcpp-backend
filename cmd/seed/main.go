package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	superadminEmail := os.Getenv("SUPERADMIN_EMAILS")
	if superadminEmail == "" {
		log.Fatalf("SUPERADMIN_EMAILS environment variable is required")
	}

	// Just take the first one if it's a comma separated list
	for i := 0; i < len(superadminEmail); i++ {
		if superadminEmail[i] == ',' {
			superadminEmail = superadminEmail[:i]
			break
		}
	}

	users := []struct {
		email    string
		username string
	}{
		{email: superadminEmail, username: "superadmin"},
		{email: "admin@example.com", username: "admin"},
		{email: "user@example.com", username: "normaluser"},
	}

	for _, u := range users {
		_, err := db.ExecContext(
			context.Background(),
			"INSERT INTO users (email, username) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING",
			u.email, u.username,
		)
		if err != nil {
			log.Printf("failed to insert user %s: %v\n", u.email, err)
		} else {
			fmt.Printf("User %s seeded successfully\n", u.email)
		}
	}

	fmt.Println("Seeding complete.")
}
