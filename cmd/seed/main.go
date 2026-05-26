package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joey/wcwcpp-backend/adapters/storage/postgres"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/core/service"
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

	ctx := context.Background()

	if err := db.PingContext(ctx); err != nil {
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
			ctx,
			"INSERT INTO users (email, username) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING",
			u.email, u.username,
		)
		if err != nil {
			log.Printf("failed to insert user %s: %v\n", u.email, err)
		} else {
			fmt.Printf("User %s seeded successfully\n", u.email)
		}
	}

	// Contest Seeding
	contestRepo := postgres.NewContestRepository(db)
	contestService := service.NewContestService(contestRepo)

	existing, err := contestRepo.GetContestBySlug(ctx, "world-cup-2026")
	if err != nil {
		log.Fatalf("failed to check for existing contest: %v", err)
	}

	if existing != nil {
		fmt.Println("Contest 'world-cup-2026' already exists. Skipping contest seeding.")
	} else {
		fmt.Println("Seeding 'world-cup-2026' contest...")

		groupConfigs := []struct {
			letter    string
			countries []string
		}{
			{"A", []string{"USA", "MEX", "CAN", "CRC"}},
			{"B", []string{"BRA", "ARG", "URU", "COL"}},
			{"C", []string{"ENG", "FRA", "GER", "ITA"}},
			{"D", []string{"ESP", "POR", "NED", "BEL"}},
			{"E", []string{"CRO", "SEN", "MAR", "TUN"}},
			{"F", []string{"JPN", "KOR", "AUS", "IRN"}},
			{"G", []string{"KSA", "QAT", "UAE", "OMN"}},
			{"H", []string{"EGY", "NGA", "GHA", "CMR"}},
			{"I", []string{"SWE", "DEN", "NOR", "FIN"}},
			{"J", []string{"SUI", "AUT", "POL", "UKR"}},
			{"K", []string{"TUR", "GRE", "CZE", "SVK"}},
			{"L", []string{"WAL", "SCO", "IRL", "NIR"}},
		}

		var groups []entity.Group
		for _, conf := range groupConfigs {
			var countries []entity.Country
			for _, code := range conf.countries {
				countries = append(countries, entity.Country{
					Code:     code,
					FullName: "Country " + code,
				})
			}
			groups = append(groups, entity.Group{
				Letter:    conf.letter,
				Countries: countries,
			})
		}

		contest := entity.Contest{
			Title:              "World Cup 2026",
			Slug:               "world-cup-2026",
			GroupUnlockDate:    time.Now().Add(-24 * time.Hour),
			GroupLockDate:      time.Now().Add(365 * 24 * time.Hour),
			KnockoutUnlockDate: time.Now().Add(-24 * time.Hour),
			KnockoutLockDate:   time.Now().Add(365 * 24 * time.Hour),
			Groups:             groups,
		}

		if err := contestService.CreateContest(ctx, contest); err != nil {
			log.Fatalf("failed to create seed contest: %v", err)
		}

		fmt.Println("Contest 'world-cup-2026' seeded successfully with 12 groups and 48 countries!")
	}

	// Dynamic DB Surgery for UAT verification of Knockout winner resolution
	fmt.Println("Performing database surgery to set up resolved knockout matches for UAT...")
	var contestID string
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'world-cup-2026'").Scan(&contestID)
	if err != nil {
		log.Fatalf("failed to get contest ID for database surgery: %v", err)
	}

	var usaID, mexID, braID, argID string
	err = db.QueryRowContext(ctx, "SELECT id FROM countries WHERE code = 'USA'").Scan(&usaID)
	if err != nil {
		log.Fatalf("failed to get USA ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM countries WHERE code = 'MEX'").Scan(&mexID)
	if err != nil {
		log.Fatalf("failed to get MEX ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM countries WHERE code = 'BRA'").Scan(&braID)
	if err != nil {
		log.Fatalf("failed to get BRA ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM countries WHERE code = 'ARG'").Scan(&argID)
	if err != nil {
		log.Fatalf("failed to get ARG ID: %v", err)
	}

	// Match index 0: USA vs MEX -> USA wins 2-1
	_, err = db.ExecContext(ctx, `
		UPDATE matches 
		SET country1_id = $1, country2_id = $2, country1_goals = 2, country2_goals = 1
		WHERE contest_id = $3 AND round = 1 AND round_index = 0
	`, usaID, mexID, contestID)
	if err != nil {
		log.Fatalf("failed to perform DB surgery on match 1: %v", err)
	}

	// Match index 1: BRA vs ARG -> ARG wins on penalties (1-1, 3-4 PKs)
	_, err = db.ExecContext(ctx, `
		UPDATE matches 
		SET country1_id = $1, country2_id = $2, country1_goals = 1, country2_goals = 1, country1_penalties = 3, country2_penalties = 4
		WHERE contest_id = $3 AND round = 1 AND round_index = 1
	`, braID, argID, contestID)
	if err != nil {
		log.Fatalf("failed to perform DB surgery on match 2: %v", err)
	}

	fmt.Println("Database surgery complete: populated 2 completed round of 16 matches successfully.")

	fmt.Println("Seeding complete.")
}
