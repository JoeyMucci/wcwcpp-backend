package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
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

	// 1. Truncate tables for a 100% clean seed execution
	fmt.Println("Truncating existing tables for a clean seed...")
	_, err = db.ExecContext(ctx, "TRUNCATE TABLE users, contests, countries CASCADE")
	if err != nil {
		log.Fatalf("failed to truncate tables: %v", err)
	}
	fmt.Println("Database tables truncated successfully.")

	// 2. Generate 50 realistic users + default users (superadmin, admin, normaluser, Joey, Joseph)
	fmt.Println("Generating user accounts...")
	var users []struct {
		email    string
		username string
	}
	users = append(users, struct{ email, username string }{email: superadminEmail, username: "superadmin"})
	users = append(users, struct{ email, username string }{email: "admin@example.com", username: "admin"})
	users = append(users, struct{ email, username string }{email: "user@example.com", username: "normaluser"})
	users = append(users, struct{ email, username string }{email: "jmucci314@gmail.com", username: "Joey Mucci"})
	users = append(users, struct{ email, username string }{email: "jpm73@njit.edu", username: "Joseph Mucci"})

	firstNames := []string{"John", "Jane", "Marcos", "Emma", "Liam", "Sophia", "Lucas", "Olivia", "Mateo", "Ava", "Gabriel", "Isabella", "David", "Mia", "Leo", "Charlotte", "Carlos", "Amelia", "Ivan", "Sofia"}
	lastNames := []string{"Smith", "Silva", "Jones", "Brown", "Davis", "Miller", "Wilson", "Moore", "Taylor", "Thomas", "Anderson", "White", "Harris", "Martin", "Thompson", "Garcia", "Martinez", "Robinson", "Clark", "Rodriguez"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 1; i <= 50; i++ {
		fn := firstNames[r.Intn(len(firstNames))]
		ln := lastNames[r.Intn(len(lastNames))]
		username := fmt.Sprintf("%s_%s_%d", strings.ToLower(fn), strings.ToLower(ln), r.Intn(100))
		email := fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(fn), strings.ToLower(ln), i)
		users = append(users, struct{ email, username string }{email: email, username: username})
	}

	userIDs := make(map[string]string) // email -> ID
	var allUserIDs []string
	for _, u := range users {
		var id string
		err := db.QueryRowContext(
			ctx,
			"INSERT INTO users (email, username) VALUES ($1, $2) RETURNING id",
			u.email, u.username,
		).Scan(&id)
		if err != nil {
			log.Fatalf("failed to insert user %s: %v\n", u.email, err)
		}
		userIDs[u.email] = id
		allUserIDs = append(allUserIDs, id)
	}
	fmt.Printf("Successfully seeded %d user accounts!\n", len(users))

	// 3. Define and Create Contests at different completion stages
	contestRepo := postgres.NewContestRepository(db)
	contestService := service.NewContestService(contestRepo)
	picksRepo := postgres.NewPicksRepository(db)

	// Contest 1: finished (Completed / Past Lock Dates / All matches played)
	contestFinished := makeContestEntity(
		"Finished",
		"finished",
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
	)

	// Contest 2: in progress (In between Group Stage and Knockout Stage)
	contestInProgress := makeContestEntity(
		"In Progress",
		"in-progress",
		time.Now().Add(-72*time.Hour),
		time.Now().Add(-24*time.Hour), // Group stage picks locked
		time.Now().Add(-24*time.Hour),
		time.Now().Add(72*time.Hour), // Knockout stage picks open
	)

	// Contest 3: future (Not started / Lock dates in the future)
	contestFuture := makeContestEntity(
		"Future",
		"future",
		time.Now().Add(24*time.Hour),
		time.Now().Add(48*time.Hour),
		time.Now().Add(72*time.Hour),
		time.Now().Add(96*time.Hour),
	)

	// Contest 4: perfect (Completed / All matches played / Joey predicts everything right)
	contestPerfect := makeContestEntity(
		"Joey Perfect Predictions",
		"perfect",
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
	)

	// Contest 5: perfect-ish (Completed / All matches played / Joey predicts everything right except third place qualifiers and third place match)
	contestPerfectIsh := makeContestEntity(
		"Joey Almost Perfect Predictions",
		"perfect-ish",
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
		time.Now().Add(-4*365*24*time.Hour),
		time.Now().Add(-3*365*24*time.Hour),
	)

	fmt.Println("Seeding Contest configurations...")
	if err := contestService.CreateContest(ctx, contestFinished); err != nil {
		log.Fatalf("failed to create Finished: %v", err)
	}
	if err := contestService.CreateContest(ctx, contestInProgress); err != nil {
		log.Fatalf("failed to create In Progress: %v", err)
	}
	if err := contestService.CreateContest(ctx, contestFuture); err != nil {
		log.Fatalf("failed to create Future: %v", err)
	}
	if err := contestService.CreateContest(ctx, contestPerfect); err != nil {
		log.Fatalf("failed to create Perfect: %v", err)
	}
	if err := contestService.CreateContest(ctx, contestPerfectIsh); err != nil {
		log.Fatalf("failed to create Perfect-ish: %v", err)
	}

	var idFinished, idInProgress, idFuture, idPerfect, idPerfectIsh string
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'finished'").Scan(&idFinished)
	if err != nil {
		log.Fatalf("failed to get finished ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'in-progress'").Scan(&idInProgress)
	if err != nil {
		log.Fatalf("failed to get in-progress ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'future'").Scan(&idFuture)
	if err != nil {
		log.Fatalf("failed to get future ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'perfect'").Scan(&idPerfect)
	if err != nil {
		log.Fatalf("failed to get perfect ID: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT id FROM contests WHERE slug = 'perfect-ish'").Scan(&idPerfectIsh)
	if err != nil {
		log.Fatalf("failed to get perfect-ish ID: %v", err)
	}

	countriesFinished, err := getContestCountries(ctx, db, idFinished)
	if err != nil {
		log.Fatalf("failed to query Finished countries: %v", err)
	}
	countriesInProgress, err := getContestCountries(ctx, db, idInProgress)
	if err != nil {
		log.Fatalf("failed to query In Progress countries: %v", err)
	}
	countriesFuture, err := getContestCountries(ctx, db, idFuture)
	if err != nil {
		log.Fatalf("failed to query Future countries: %v", err)
	}
	countriesPerfect, err := getContestCountries(ctx, db, idPerfect)
	if err != nil {
		log.Fatalf("failed to query Perfect countries: %v", err)
	}
	countriesPerfectIsh, err := getContestCountries(ctx, db, idPerfectIsh)
	if err != nil {
		log.Fatalf("failed to query Perfect-ish countries: %v", err)
	}

	allLetters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	joeyID := userIDs["jmucci314@gmail.com"]
	jpmID := userIDs["jpm73@njit.edu"]

	// ==========================================
	// 4. Seed Contest: Future
	// ==========================================
	fmt.Println("Seeding predictions for 'future' contest...")
	// For future, we only seed group picks since knockout matchups are TBD.
	for _, uID := range allUserIDs {
		if err := seedGroupPicksForUser(ctx, picksRepo, db, uID, idFuture, countriesFuture, allLetters, r); err != nil {
			log.Fatalf("failed seeding group picks for future: %v", err)
		}
	}

	// ==========================================
	// 5. Seed Contest: In Progress
	// ==========================================
	fmt.Println("Seeding predictions for 'in-progress' contest...")
	// A. Seed group picks first for all users
	for _, uID := range allUserIDs {
		if err := seedGroupPicksForUser(ctx, picksRepo, db, uID, idInProgress, countriesInProgress, allLetters, r); err != nil {
			log.Fatalf("failed seeding group picks for in-progress: %v", err)
		}
	}
	// B. Play group stage matches and finalize group standings/qualifiers
	fmt.Println("Simulating and playing out 'in-progress' group stage...")
	simulateGroupStage(ctx, contestRepo, db, idInProgress, r)

	// C. Now that Round of 32 knockout matches are populated, seed knockout picks for all users
	fmt.Println("Seeding knockout predictions for 'in-progress'...")
	for _, uID := range allUserIDs {
		if err := seedKnockoutPicksForUserWithOfficialMatchups(ctx, picksRepo, db, uID, idInProgress, r); err != nil {
			log.Fatalf("failed seeding knockout picks for in-progress: %v", err)
		}
	}

	// ==========================================
	// 6. Seed Contest: Finished
	// ==========================================
	fmt.Println("Seeding predictions for 'finished' contest...")
	// A. Seed group picks first for all users
	for _, uID := range allUserIDs {
		if err := seedGroupPicksForUser(ctx, picksRepo, db, uID, idFinished, countriesFinished, allLetters, r); err != nil {
			log.Fatalf("failed seeding group picks for finished: %v", err)
		}
	}
	// B. Play group stage matches and finalize group standings/qualifiers
	fmt.Println("Simulating and playing out 'finished' group stage...")
	simulateGroupStage(ctx, contestRepo, db, idFinished, r)

	// C. Now that Round of 32 knockout matches are populated, seed knockout picks for all users
	fmt.Println("Seeding knockout predictions for 'finished'...")
	for _, uID := range allUserIDs {
		if err := seedKnockoutPicksForUserWithOfficialMatchups(ctx, picksRepo, db, uID, idFinished, r); err != nil {
			log.Fatalf("failed seeding knockout picks for finished: %v", err)
		}
	}
	// D. Play knockout stage matches to determine winners, which will calculate/update knockout points
	fmt.Println("Simulating and playing out 'finished' knockout stage...")
	simulateKnockoutStage(ctx, contestRepo, db, idFinished, r)

	// ==========================================
	// 6.5. Seed Contest: Perfect
	// ==========================================
	fmt.Println("Seeding predictions for 'perfect' contest...")
	// A. Seed group picks first for all users
	for _, uID := range allUserIDs {
		if uID == joeyID {
			if err := seedGroupPicksForJoey(ctx, picksRepo, db, uID, idPerfect, countriesPerfect, allLetters, true); err != nil {
				log.Fatalf("failed seeding group picks for perfect (Joey): %v", err)
			}
		} else {
			if err := seedGroupPicksForUser(ctx, picksRepo, db, uID, idPerfect, countriesPerfect, allLetters, r); err != nil {
				log.Fatalf("failed seeding group picks for perfect: %v", err)
			}
		}
	}
	// B. Play group stage matches and finalize group standings/qualifiers
	fmt.Println("Simulating and playing out 'perfect' group stage...")
	simulateGroupStageDeterministic(ctx, contestRepo, db, idPerfect)

	// C. Now that Round of 32 knockout matches are populated, seed knockout picks for all users
	fmt.Println("Seeding knockout predictions for 'perfect'...")
	for _, uID := range allUserIDs {
		if uID == joeyID {
			if err := seedKnockoutPicksForJoey(ctx, picksRepo, db, uID, idPerfect, true); err != nil {
				log.Fatalf("failed seeding knockout picks for perfect (Joey): %v", err)
			}
		} else {
			if err := seedKnockoutPicksForUserWithOfficialMatchups(ctx, picksRepo, db, uID, idPerfect, r); err != nil {
				log.Fatalf("failed seeding knockout picks for perfect: %v", err)
			}
		}
	}
	// D. Play knockout stage matches to determine winners, which will calculate/update knockout points
	fmt.Println("Simulating and playing out 'perfect' knockout stage...")
	simulateKnockoutStageDeterministic(ctx, contestRepo, db, idPerfect)

	// ==========================================
	// 6.6. Seed Contest: Perfect-ish
	// ==========================================
	fmt.Println("Seeding predictions for 'perfect-ish' contest...")
	// A. Seed group picks first for all users
	for _, uID := range allUserIDs {
		if uID == joeyID {
			if err := seedGroupPicksForJoey(ctx, picksRepo, db, uID, idPerfectIsh, countriesPerfectIsh, allLetters, false); err != nil {
				log.Fatalf("failed seeding group picks for perfect-ish (Joey): %v", err)
			}
		} else {
			if err := seedGroupPicksForUser(ctx, picksRepo, db, uID, idPerfectIsh, countriesPerfectIsh, allLetters, r); err != nil {
				log.Fatalf("failed seeding group picks for perfect-ish: %v", err)
			}
		}
	}
	// B. Play group stage matches and finalize group standings/qualifiers
	fmt.Println("Simulating and playing out 'perfect-ish' group stage...")
	simulateGroupStageDeterministic(ctx, contestRepo, db, idPerfectIsh)

	// C. Now that Round of 32 knockout matches are populated, seed knockout picks for all users
	fmt.Println("Seeding knockout predictions for 'perfect-ish'...")
	for _, uID := range allUserIDs {
		if uID == joeyID {
			if err := seedKnockoutPicksForJoey(ctx, picksRepo, db, uID, idPerfectIsh, false); err != nil {
				log.Fatalf("failed seeding knockout picks for perfect-ish (Joey): %v", err)
			}
		} else {
			if err := seedKnockoutPicksForUserWithOfficialMatchups(ctx, picksRepo, db, uID, idPerfectIsh, r); err != nil {
				log.Fatalf("failed seeding knockout picks for perfect-ish: %v", err)
			}
		}
	}
	// D. Play knockout stage matches to determine winners, which will calculate/update knockout points
	fmt.Println("Simulating and playing out 'perfect-ish' knockout stage...")
	simulateKnockoutStageDeterministic(ctx, contestRepo, db, idPerfectIsh)

	// ==========================================
	// 7. Seed Subcontests (Mini-leagues / Pools)
	// ==========================================
	fmt.Println("Seeding Subcontests and mini-leagues...")

	// Subcontests for In Progress
	seedSubcontest(ctx, db, idInProgress, userIDs[superadminEmail], "Antigravity Global In Progress", "antigravity-global-in-progress", "JOINPROG", append(allUserIDs[1:28], joeyID, jpmID))
	seedSubcontest(ctx, db, idInProgress, userIDs["admin@example.com"], "Engineering Side In Progress", "engineering-side-in-progress", "DEVPROG", append(allUserIDs[10:35], joeyID, jpmID))
	seedSubcontest(ctx, db, idInProgress, joeyID, "Joey's In Progress Club", "joeys-in-progress-club", "JOEYPROG", []string{userIDs["admin@example.com"], jpmID, allUserIDs[5], allUserIDs[6]})
	seedSubcontest(ctx, db, idInProgress, jpmID, "NJIT In Progress Arena", "njit-in-progress-arena", "NJITPROG", []string{userIDs["admin@example.com"], joeyID, allUserIDs[7], allUserIDs[8]})

	// Subcontests for Finished
	seedSubcontest(ctx, db, idFinished, userIDs[superadminEmail], "Finished Legacy Cup", "finished-legacy-cup", "HISTDONE", allUserIDs)
	seedSubcontest(ctx, db, idFinished, joeyID, "Joey's Finished Lounge", "joeys-finished-lounge", "JOEYDONE", []string{userIDs["admin@example.com"], jpmID, allUserIDs[5], allUserIDs[6]})

	// Subcontests for Future
	seedSubcontest(ctx, db, idFuture, userIDs[superadminEmail], "Future Global Arena", "future-global-arena", "FUTGLOBA", append(allUserIDs[1:28], joeyID, jpmID))
	seedSubcontest(ctx, db, idFuture, jpmID, "NJIT Future Pool", "njit-future-pool", "NUTFUTUR", []string{userIDs["admin@example.com"], joeyID, allUserIDs[7], allUserIDs[8]})

	// Subcontests for Perfect
	seedSubcontest(ctx, db, idPerfect, userIDs[superadminEmail], "Perfect Global Arena", "perfect-global-arena", "PERFCODE", append(allUserIDs[1:28], joeyID, jpmID))

	// Subcontests for Perfect-ish
	seedSubcontest(ctx, db, idPerfectIsh, userIDs[superadminEmail], "Almost Perfect Global Arena", "almost-perfect-global-arena", "ALMPERFC", append(allUserIDs[1:28], joeyID, jpmID))

	fmt.Println("\n=============================================")
	fmt.Println("DATABASE SEEDING COMPLETED SUCCESSFULLY!")
	fmt.Printf(" - Seeded %d active users\n", len(users))
	fmt.Printf(" - Seeding Contest 'finished' (Completed / Full points populated)\n")
	fmt.Printf(" - Seeding Contest 'in-progress' (Group stage played/finalized / Knockout picks made / Scores calculated)\n")
	fmt.Printf(" - Seeding Contest 'future' (Group picks submitted / Locked dates in future)\n")
	fmt.Printf(" - Seeding Contest 'perfect' (Completed / Joey predicts everything right)\n")
	fmt.Printf(" - Seeding Contest 'perfect-ish' (Completed / Joey predicts everything right except third place qualifiers and third place match)\n")
	fmt.Printf(" - Created competitive subleague pools with leaderboards populated for all contests\n")
	fmt.Println("=============================================")

	// Print exact predictions and results
	printJmucciPicks(ctx, db, joeyID)
	printContestResults(ctx, db)
}

// Helpers

var countryNames = map[string]string{
	"USA": "United States",
	"MEX": "Mexico",
	"CAN": "Canada",
	"CRC": "Costa Rica",
	"BRA": "Brazil",
	"ARG": "Argentina",
	"URU": "Uruguay",
	"COL": "Colombia",
	"ENG": "England",
	"FRA": "France",
	"GER": "Germany",
	"ITA": "Italy",
	"ESP": "Spain",
	"POR": "Portugal",
	"NED": "Netherlands",
	"BEL": "Belgium",
	"CRO": "Croatia",
	"SEN": "Senegal",
	"MAR": "Morocco",
	"TUN": "Tunisia",
	"JPN": "Japan",
	"KOR": "South Korea",
	"AUS": "Australia",
	"IRN": "Iran",
	"KSA": "Saudi Arabia",
	"QAT": "Qatar",
	"UAE": "United Arab Emirates",
	"OMN": "Oman",
	"EGY": "Egypt",
	"NGA": "Nigeria",
	"GHA": "Ghana",
	"CMR": "Cameroon",
	"SWE": "Sweden",
	"DEN": "Denmark",
	"NOR": "Norway",
	"FIN": "Finland",
	"SUI": "Switzerland",
	"AUT": "Austria",
	"POL": "Poland",
	"UKR": "Ukraine",
	"TUR": "Turkey",
	"GRE": "Greece",
	"CZE": "Czech Republic",
	"SVK": "Slovakia",
	"WAL": "Wales",
	"SCO": "Scotland",
	"IRL": "Ireland",
	"NIR": "Northern Ireland",
}

func makeContestEntity(title, slug string, groupUnlock, groupLock, koUnlock, koLock time.Time) entity.Contest {
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
			name, found := countryNames[code]
			if !found {
				name = "Country " + code
			}
			countries = append(countries, entity.Country{
				Code:     code,
				FullName: name,
			})
		}
		groups = append(groups, entity.Group{
			Letter:    conf.letter,
			Countries: countries,
		})
	}

	return entity.Contest{
		Title:              title,
		Slug:               slug,
		GroupUnlockDate:    groupUnlock,
		GroupLockDate:      groupLock,
		KnockoutUnlockDate: koUnlock,
		KnockoutLockDate:   koLock,
		Groups:             groups,
	}
}

func getContestCountries(ctx context.Context, db *sql.DB, contestID string) ([]entity.Country, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT c.code, c.full_name 
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1
	`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []entity.Country
	for rows.Next() {
		var c entity.Country
		if err := rows.Scan(&c.Code, &c.FullName); err != nil {
			return nil, err
		}
		countries = append(countries, c)
	}
	return countries, nil
}

func seedGroupPicksForUser(ctx context.Context, picksRepo *postgres.PicksRepository, db *sql.DB, userID, contestID string, countries []entity.Country, letters []string, r *rand.Rand) error {
	rows, err := db.QueryContext(ctx, `
		SELECT c.code, gs.letter 
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1
	`, contestID)
	if err != nil {
		return err
	}
	defer rows.Close()

	groupMap := make(map[string][]entity.Country)
	for rows.Next() {
		var code, letter string
		if err := rows.Scan(&code, &letter); err != nil {
			return err
		}
		var matchCountry entity.Country
		for _, c := range countries {
			if c.Code == code {
				matchCountry = c
				break
			}
		}
		groupMap[letter] = append(groupMap[letter], matchCountry)
	}

	var picks []entity.GroupPick
	for _, letter := range letters {
		groupCountries, ok := groupMap[letter]
		if !ok || len(groupCountries) == 0 {
			continue
		}

		shuffled := make([]entity.Country, len(groupCountries))
		copy(shuffled, groupCountries)
		r.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		var entries []entity.GroupPickEntry
		for idx, t := range shuffled {
			entries = append(entries, entity.GroupPickEntry{
				Country: t,
				Place:   idx + 1,
			})
		}

		picks = append(picks, entity.GroupPick{
			Letter:         letter,
			Entries:        entries,
			ExtraQualifier: r.Intn(2) == 0,
		})
	}

	return picksRepo.CreateGroupPicks(ctx, userID, contestID, picks)
}

func seedKnockoutPicksForUserWithOfficialMatchups(ctx context.Context, picksRepo *postgres.PicksRepository, db *sql.DB, userID, contestID string, r *rand.Rand) error {
	// 1. Query the 16 Round of 32 matches (Round = 1) from the database
	rows, err := db.QueryContext(ctx, `
		SELECT m.round_index, c1.code, c1.full_name, c2.code, c2.full_name
		FROM matches m
		JOIN countries c1 ON m.country1_id = c1.id
		JOIN countries c2 ON m.country2_id = c2.id
		WHERE m.contest_id = $1 AND m.round = 1
		ORDER BY m.round_index ASC
	`, contestID)
	if err != nil {
		return fmt.Errorf("failed to query Round of 32 matches: %w", err)
	}
	defer rows.Close()

	type r32Match struct {
		idx    int
		c1, c2 entity.Country
	}
	var r32Matches []r32Match
	for rows.Next() {
		var rm r32Match
		if err := rows.Scan(&rm.idx, &rm.c1.Code, &rm.c1.FullName, &rm.c2.Code, &rm.c2.FullName); err != nil {
			return fmt.Errorf("failed to scan Round of 32 match: %w", err)
		}
		r32Matches = append(r32Matches, rm)
	}

	if len(r32Matches) != 16 {
		return fmt.Errorf("expected 16 Round of 32 matches, found %d", len(r32Matches))
	}

	// 2. Select winners of Round of 32 (Round 1) to advance to Round of 16 (Round 2)
	var entries []entity.KnockoutPickEntry
	var r16Picks []entity.Country
	for _, m := range r32Matches {
		winner := m.c1
		if r.Intn(2) == 0 {
			winner = m.c2
		}
		r16Picks = append(r16Picks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 2})
	}

	// 3. Select winners of Round of 16 (Round 2) to advance to Quarterfinals (Round 3)
	var qfPicks []entity.Country
	for i := 0; i < 8; i++ {
		winner := r16Picks[i*2]
		if r.Intn(2) == 0 {
			winner = r16Picks[i*2+1]
		}
		qfPicks = append(qfPicks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 3})
	}

	// 4. Select winners of Quarterfinals (Round 3) to advance to Semifinals (Round 4)
	var sfPicks []entity.Country
	for i := 0; i < 4; i++ {
		winner := qfPicks[i*2]
		if r.Intn(2) == 0 {
			winner = qfPicks[i*2+1]
		}
		sfPicks = append(sfPicks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 4})
	}

	// 5. Select winners of Semifinals (Round 4) to advance to Finals (Round 5)
	var finalPicks []entity.Country
	var sfLosers []entity.Country
	for i := 0; i < 2; i++ {
		t1 := sfPicks[i*2]
		t2 := sfPicks[i*2+1]
		winner := t1
		loser := t2
		if r.Intn(2) == 0 {
			winner = t2
			loser = t1
		}
		finalPicks = append(finalPicks, winner)
		sfLosers = append(sfLosers, loser)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 5})
	}

	// 6. Select Champion (Round 6) from the Finals
	champion := finalPicks[0]
	if r.Intn(2) == 0 {
		champion = finalPicks[1]
	}
	entries = append(entries, entity.KnockoutPickEntry{Country: champion, Round: 6})

	// 7. Select Third Place Winner (Round 7) from the Semifinals Losers
	thirdPlaceWinner := sfLosers[0]
	if r.Intn(2) == 0 {
		thirdPlaceWinner = sfLosers[1]
	}
	entries = append(entries, entity.KnockoutPickEntry{Country: thirdPlaceWinner, Round: 7})

	pick := entity.KnockoutPick{Entries: entries}
	return picksRepo.CreateKnockoutPicks(ctx, userID, contestID, pick)
}

func simulateGroupStage(ctx context.Context, contestRepo *postgres.ContestRepository, db *sql.DB, contestID string, r *rand.Rand) {
	// 1. Play Group Stage matches (Round = 0)
	rows, err := db.QueryContext(ctx, `
		SELECT m.id, c1.code, c2.code 
		FROM matches m
		JOIN countries c1 ON m.country1_id = c1.id
		JOIN countries c2 ON m.country2_id = c2.id
		WHERE m.contest_id = $1 AND m.round = 0
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query group matches: %v", err)
	}

	type matchInfo struct {
		id, c1, c2 string
	}
	var groupMatches []matchInfo
	for rows.Next() {
		var m matchInfo
		if err := rows.Scan(&m.id, &m.c1, &m.c2); err != nil {
			log.Fatalf("failed to scan group match: %v", err)
		}
		groupMatches = append(groupMatches, m)
	}
	rows.Close()

	for _, gm := range groupMatches {
		g1 := r.Intn(4)
		g2 := r.Intn(4)
		match := entity.Match{
			Country1:      &entity.Country{Code: gm.c1},
			Country2:      &entity.Country{Code: gm.c2},
			Country1Goals: &g1,
			Country2Goals: &g2,
			Round:         0,
		}
		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to play group match: %v", err)
		}
	}

	// 2. Finalize Group Standings (A-L)
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	for _, letter := range letters {
		orderRows, err := db.QueryContext(ctx, `
			SELECT c.code 
			FROM group_standings gs
			JOIN countries c ON gs.country_id = c.id
			WHERE gs.contest_id = $1 AND gs.letter = $2
			ORDER BY gs.points DESC, gs.gd DESC, gs.gf DESC, c.code ASC
		`, contestID, letter)
		if err != nil {
			log.Fatalf("failed to query order for group %s: %v", letter, err)
		}

		var orderedCodes []string
		for orderRows.Next() {
			var code string
			if err := orderRows.Scan(&code); err != nil {
				log.Fatalf("failed to scan country code: %v", err)
			}
			orderedCodes = append(orderedCodes, code)
		}
		orderRows.Close()

		if err := contestRepo.FinalizeGroupRankings(ctx, contestID, letter, orderedCodes); err != nil {
			log.Fatalf("failed to finalize rankings for group %s: %v", letter, err)
		}
	}

	// 3. Finalize Third Place Qualifiers dynamically based on actual standings
	thirdRows, err := db.QueryContext(ctx, `
		SELECT gs.letter
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1 AND gs.rank = 3
		ORDER BY gs.points DESC, gs.gd DESC, gs.gf DESC, c.code ASC
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query third place standings: %v", err)
	}

	var sortedThirdPlaceLetters []string
	for thirdRows.Next() {
		var letter string
		if err := thirdRows.Scan(&letter); err != nil {
			log.Fatalf("failed to scan third place letter: %v", err)
		}
		sortedThirdPlaceLetters = append(sortedThirdPlaceLetters, letter)
	}
	thirdRows.Close()

	if len(sortedThirdPlaceLetters) != 12 {
		log.Fatalf("expected exactly 12 third-place teams, got %d", len(sortedThirdPlaceLetters))
	}

	qualifierMap := make(map[string]bool)
	for i, letter := range sortedThirdPlaceLetters {
		qualifierMap[letter] = (i < 8)
	}

	for _, letter := range letters {
		isQualifier := qualifierMap[letter]
		if err := contestRepo.FinalizeThirdPlaceQualifier(ctx, contestID, letter, isQualifier); err != nil {
			log.Fatalf("failed to finalize third place qualifier for group %s: %v", letter, err)
		}
	}

	// 4. Set up Round of 32 knockout matches (Round = 1) without goals or penalties
	advRows, err := db.QueryContext(ctx, `
		SELECT c.code 
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1 AND (gs.rank = 1 OR gs.rank = 2 OR (gs.rank = 3 AND gs.is_third_place_qualifier = true))
		ORDER BY gs.letter ASC, gs.rank ASC
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query advancing teams: %v", err)
	}

	var advancingTeams []string
	for advRows.Next() {
		var code string
		if err := advRows.Scan(&code); err != nil {
			log.Fatalf("failed to scan advancing team code: %v", err)
		}
		advancingTeams = append(advancingTeams, code)
	}
	advRows.Close()

	if len(advancingTeams) != 32 {
		log.Fatalf("expected exactly 32 advancing teams, got %d", len(advancingTeams))
	}

	for i := 0; i < 16; i++ {
		c1 := advancingTeams[2*i]
		c2 := advancingTeams[2*i+1]

		matchIdx := i
		match := entity.Match{
			Round:      1,
			RoundIndex: &matchIdx,
			Country1:   &entity.Country{Code: c1},
			Country2:   &entity.Country{Code: c2},
		}

		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to update Round of 32 match index %d: %v", i, err)
		}
	}
}

func simulateKnockoutStage(ctx context.Context, contestRepo *postgres.ContestRepository, db *sql.DB, contestID string, r *rand.Rand) {
	// Play Round of 32 knockout matches (Round = 1) by adding goals and penalties
	for i := 0; i < 16; i++ {
		g1 := r.Intn(4)
		g2 := r.Intn(4)
		var p1, p2 *int
		if g1 == g2 {
			p1Val := 4
			p2Val := 3
			p1 = &p1Val
			p2 = &p2Val
		}

		matchIdx := i
		match := entity.Match{
			Round:             1,
			RoundIndex:        &matchIdx,
			Country1Goals:     &g1,
			Country2Goals:     &g2,
			Country1Penalties: p1,
			Country2Penalties: p2,
		}

		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to play Round of 32 match index %d: %v", i, err)
		}
	}

	// Play subsequent rounds (Rounds 2 to 5)
	for round := 2; round <= 5; round++ {
		matchRows, err := db.QueryContext(ctx, `
			SELECT m.round_index, c1.code, c2.code 
			FROM matches m
			JOIN countries c1 ON m.country1_id = c1.id
			JOIN countries c2 ON m.country2_id = c2.id
			WHERE m.contest_id = $1 AND m.round = $2
			ORDER BY m.round_index ASC
		`, contestID, round)
		if err != nil {
			log.Fatalf("failed to query Round %d matches: %v", round, err)
		}

		type koMatch struct {
			idx    int
			c1, c2 string
		}
		var koMatches []koMatch
		for matchRows.Next() {
			var km koMatch
			if err := matchRows.Scan(&km.idx, &km.c1, &km.c2); err != nil {
				log.Fatalf("failed to scan Round %d match: %v", round, err)
			}
			koMatches = append(koMatches, km)
		}
		matchRows.Close()

		for _, km := range koMatches {
			g1 := r.Intn(4)
			g2 := r.Intn(4)
			var p1, p2 *int
			if g1 == g2 {
				p1Val := 4
				p2Val := 3
				p1 = &p1Val
				p2 = &p2Val
			}

			mIdx := km.idx
			match := entity.Match{
				Round:             round,
				RoundIndex:        &mIdx,
				Country1Goals:     &g1,
				Country2Goals:     &g2,
				Country1Penalties: p1,
				Country2Penalties: p2,
			}

			if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
				log.Fatalf("failed to play Round %d match index %d: %v", round, km.idx, err)
			}
		}
	}
}

func seedSubcontest(ctx context.Context, db *sql.DB, contestID, ownerUserID, title, slug, joinCode string, memberIDs []string) {
	var subID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO subcontests (contest_id, user_id, join_code, title, slug)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, contestID, ownerUserID, joinCode, title, slug).Scan(&subID)
	if err != nil {
		log.Fatalf("failed to insert subcontest %s: %v", title, err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO subcontest_entries (subcontest_id, user_id)
		VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, subID, ownerUserID)
	if err != nil {
		log.Fatalf("failed to add owner to subcontest %s: %v", title, err)
	}

	for _, mID := range memberIDs {
		_, err = db.ExecContext(ctx, `
			INSERT INTO subcontest_entries (subcontest_id, user_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, subID, mID)
		if err != nil {
			log.Fatalf("failed to add member %s to subcontest %s: %v", mID, title, err)
		}
	}
}

func printJmucciPicks(ctx context.Context, db *sql.DB, userID string) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.id, c.title, c.slug
		FROM contests c
		ORDER BY c.slug ASC
	`)
	if err != nil {
		log.Fatalf("failed to query contests for picks printing: %v", err)
	}
	defer rows.Close()

	type contestInfo struct {
		id, title, slug string
	}
	var contests []contestInfo
	for rows.Next() {
		var ci contestInfo
		if err := rows.Scan(&ci.id, &ci.title, &ci.slug); err != nil {
			log.Fatalf("failed to scan contest: %v", err)
		}
		contests = append(contests, ci)
	}

	fmt.Println("\n==================================================================")
	fmt.Println("   PREDICTION REPORT FOR USER: jmucci314@gmail.com (Joey Mucci)")
	fmt.Println("==================================================================")

	for _, ci := range contests {
		fmt.Printf("\nContest: %s (%s)\n", ci.title, ci.slug)
		fmt.Println("------------------------------------------------------------------")

		// 1. Query group picks
		gpRows, err := db.QueryContext(ctx, `
			SELECT gp.letter, gp.place, gp.extra_qualifier, c.code, c.full_name
			FROM group_picks gp
			JOIN countries c ON gp.country_id = c.id
			WHERE gp.user_id = $1 AND gp.contest_id = $2
			ORDER BY gp.letter ASC, gp.place ASC
		`, userID, ci.id)
		if err != nil {
			log.Fatalf("failed to query group picks: %v", err)
		}

		type groupPick struct {
			letter          string
			place           int
			extraQualifier  bool
			countryCode     string
			countryFullName string
		}
		var groupPicks []groupPick
		for gpRows.Next() {
			var gp groupPick
			if err := gpRows.Scan(&gp.letter, &gp.place, &gp.extraQualifier, &gp.countryCode, &gp.countryFullName); err != nil {
				log.Fatalf("failed to scan group pick: %v", err)
			}
			groupPicks = append(groupPicks, gp)
		}
		gpRows.Close()

		if len(groupPicks) > 0 {
			fmt.Println("  [Group Stage Picks]")
			currentLetter := ""
			for _, gp := range groupPicks {
				if gp.letter != currentLetter {
					if currentLetter != "" {
						fmt.Println()
					}
					fmt.Printf("    Group %s: ", gp.letter)
					currentLetter = gp.letter
				} else {
					fmt.Print(" > ")
				}
				fmt.Printf("%d. %s (%s)", gp.place, gp.countryFullName, gp.countryCode)
				if gp.extraQualifier {
					fmt.Print(" [3rd-place Q]")
				}
			}
			fmt.Println("")
		} else {
			fmt.Println("  [Group Stage Picks] None submitted yet.")
		}

		// 2. Query knockout picks
		kpRows, err := db.QueryContext(ctx, `
			SELECT kp.round, c.code, c.full_name
			FROM knockout_picks kp
			JOIN countries c ON kp.country_id = c.id
			WHERE kp.user_id = $1 AND kp.contest_id = $2
			ORDER BY kp.round ASC, c.full_name ASC
		`, userID, ci.id)
		if err != nil {
			log.Fatalf("failed to query knockout picks: %v", err)
		}

		type knockoutPick struct {
			round           int
			countryCode     string
			countryFullName string
		}
		var knockoutPicks []knockoutPick
		for kpRows.Next() {
			var kp knockoutPick
			if err := kpRows.Scan(&kp.round, &kp.countryCode, &kp.countryFullName); err != nil {
				log.Fatalf("failed to scan knockout pick: %v", err)
			}
			knockoutPicks = append(knockoutPicks, kp)
		}
		kpRows.Close()

		if len(knockoutPicks) > 0 {
			fmt.Println("  [Knockout Stage Picks]")
			// Group by round
			picksByRound := make(map[int][]string)
			for _, kp := range knockoutPicks {
				picksByRound[kp.round] = append(picksByRound[kp.round], fmt.Sprintf("%s (%s)", kp.countryFullName, kp.countryCode))
			}

			rounds := []struct {
				val  int
				name string
			}{
				{2, "Round of 16"},
				{3, "Quarterfinals"},
				{4, "Semifinals"},
				{5, "Finals"},
				{6, "Champion"},
				{7, "Third Place Winner"},
			}

			for _, rInfo := range rounds {
				list := picksByRound[rInfo.val]
				if len(list) > 0 {
					fmt.Printf("    %-30s: %s\n", rInfo.name, strings.Join(list, ", "))
				}
			}
		} else {
			fmt.Println("  [Knockout Stage Picks] None submitted yet.")
		}
	}
	fmt.Println("==================================================================")
}

func printContestResults(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.id, c.title, c.slug
		FROM contests c
		ORDER BY c.slug ASC
	`)
	if err != nil {
		log.Fatalf("failed to query contests for results printing: %v", err)
	}
	defer rows.Close()

	type contestInfo struct {
		id, title, slug string
	}
	var contests []contestInfo
	for rows.Next() {
		var ci contestInfo
		if err := rows.Scan(&ci.id, &ci.title, &ci.slug); err != nil {
			log.Fatalf("failed to scan contest: %v", err)
		}
		contests = append(contests, ci)
	}

	fmt.Println("\n==================================================================")
	fmt.Println("                    OFFICIAL CONTEST RESULTS REPORT")
	fmt.Println("==================================================================")

	for _, ci := range contests {
		fmt.Printf("\nContest: %s (%s)\n", ci.title, ci.slug)
		fmt.Println("------------------------------------------------------------------")

		// 1. Print official group standings
		gsRows, err := db.QueryContext(ctx, `
			SELECT gs.letter, gs.rank, gs.points, gs.wins, gs.draws, gs.losses, gs.gf, gs.ga, gs.gd, gs.cs, gs.is_third_place_qualifier, c.code, c.full_name
			FROM group_standings gs
			JOIN countries c ON gs.country_id = c.id
			WHERE gs.contest_id = $1
			ORDER BY gs.letter ASC, gs.rank ASC
		`, ci.id)
		if err != nil {
			log.Fatalf("failed to query group standings: %v", err)
		}

		type standingInfo struct {
			letter                      string
			rank                        int
			points, wins, draws, losses int
			gf, ga, gd, cs              int
			isThirdPlaceQ               bool
			countryCode                 string
			countryFullName             string
		}
		var standings []standingInfo
		for gsRows.Next() {
			var si standingInfo
			var rankNull sql.NullInt64
			var qNull sql.NullBool
			if err := gsRows.Scan(&si.letter, &rankNull, &si.points, &si.wins, &si.draws, &si.losses, &si.gf, &si.ga, &si.gd, &si.cs, &qNull, &si.countryCode, &si.countryFullName); err != nil {
				log.Fatalf("failed to scan standing: %v", err)
			}
			if rankNull.Valid {
				si.rank = int(rankNull.Int64)
			}
			if qNull.Valid {
				si.isThirdPlaceQ = qNull.Bool
			}
			standings = append(standings, si)
		}
		gsRows.Close()

		if len(standings) > 0 {
			fmt.Println("  [Official Group Stage Standings]")
			currentLetter := ""
			for _, si := range standings {
				if si.letter != currentLetter {
					fmt.Printf("    Group %s Standings:\n", si.letter)
					fmt.Printf("      %-4s %-25s %-3s %-2s %-2s %-2s %-3s %-3s %-3s %-3s %s\n", "Rank", "Country", "Pts", "W", "D", "L", "GF", "GA", "GD", "CS", "Status")
					currentLetter = si.letter
				}
				status := ""
				if si.rank <= 2 {
					status = "Q (Top 2)"
				} else if si.isThirdPlaceQ {
					status = "Q (Best 3rd)"
				}
				fmt.Printf("      %-4d %-25s %-3d %-2d %-2d %-2d %-3d %-3d %-3d %-3d %s\n",
					si.rank, fmt.Sprintf("%s (%s)", si.countryFullName, si.countryCode),
					si.points, si.wins, si.draws, si.losses, si.gf, si.ga, si.gd, si.cs, status)
			}
			fmt.Println()
		} else {
			fmt.Println("  [Official Group Stage Standings] No standings computed yet.")
		}

		// 2. Print official Knockout Matches
		mRows, err := db.QueryContext(ctx, `
			SELECT m.round, m.round_index, c1.full_name, c1.code, c2.full_name, c2.code, m.country1_goals, m.country2_goals, m.country1_penalties, m.country2_penalties
			FROM matches m
			LEFT JOIN countries c1 ON m.country1_id = c1.id
			LEFT JOIN countries c2 ON m.country2_id = c2.id
			WHERE m.contest_id = $1 AND m.round > 0
			ORDER BY m.round ASC, m.round_index ASC
		`, ci.id)
		if err != nil {
			log.Fatalf("failed to query matches: %v", err)
		}

		type matchResult struct {
			round          int
			roundIndex     int
			c1Name, c1Code string
			c2Name, c2Code string
			g1, g2         sql.NullInt64
			p1, p2         sql.NullInt64
		}
		var matches []matchResult
		for mRows.Next() {
			var mr matchResult
			var idxNull sql.NullInt64
			var c1NameNull, c1CodeNull, c2NameNull, c2CodeNull sql.NullString
			if err := mRows.Scan(&mr.round, &idxNull, &c1NameNull, &c1CodeNull, &c2NameNull, &c2CodeNull, &mr.g1, &mr.g2, &mr.p1, &mr.p2); err != nil {
				log.Fatalf("failed to scan match: %v", err)
			}
			if idxNull.Valid {
				mr.roundIndex = int(idxNull.Int64)
			}
			if c1NameNull.Valid {
				mr.c1Name = c1NameNull.String
			}
			if c1CodeNull.Valid {
				mr.c1Code = c1CodeNull.String
			}
			if c2NameNull.Valid {
				mr.c2Name = c2NameNull.String
			}
			if c2CodeNull.Valid {
				mr.c2Code = c2CodeNull.String
			}
			matches = append(matches, mr)
		}
		mRows.Close()

		if len(matches) > 0 {
			fmt.Println("  [Official Knockout Matches]")
			rounds := map[int]string{
				1: "Round of 32",
				2: "Round of 16",
				3: "Quarterfinals",
				4: "Semifinals",
				5: "Grand Finals & Third Place Match",
			}

			currentRound := -1
			for _, m := range matches {
				if m.round != currentRound {
					fmt.Printf("    %s:\n", rounds[m.round])
					currentRound = m.round
				}
				scoreStr := "vs"
				if m.g1.Valid && m.g2.Valid {
					scoreStr = fmt.Sprintf("%d - %d", m.g1.Int64, m.g2.Int64)
					if m.p1.Valid && m.p2.Valid {
						scoreStr = fmt.Sprintf("%s (%d - %d Pens)", scoreStr, m.p1.Int64, m.p2.Int64)
					}
				}
				team1 := fmt.Sprintf("%s (%s)", m.c1Name, m.c1Code)
				if m.c1Name == "" {
					team1 = "TBD"
				}
				team2 := fmt.Sprintf("%s (%s)", m.c2Name, m.c2Code)
				if m.c2Name == "" {
					team2 = "TBD"
				}
				fmt.Printf("      Match %-2d: %-35s %-20s %-35s\n", m.roundIndex, team1, scoreStr, team2)
			}
		} else {
			fmt.Println("  [Official Knockout Matches] No knockout matches played yet.")
		}
	}
	fmt.Println("==================================================================")
}

var countryIndexInGroup = map[string]int{
	// Group A
	"USA": 0, "MEX": 1, "CAN": 2, "CRC": 3,
	// Group B
	"BRA": 0, "ARG": 1, "URU": 2, "COL": 3,
	// Group C
	"ENG": 0, "FRA": 1, "GER": 2, "ITA": 3,
	// Group D
	"ESP": 0, "POR": 1, "NED": 2, "BEL": 3,
	// Group E
	"CRO": 0, "SEN": 1, "MAR": 2, "TUN": 3,
	// Group F
	"JPN": 0, "KOR": 1, "AUS": 2, "IRN": 3,
	// Group G
	"KSA": 0, "QAT": 1, "UAE": 2, "OMN": 3,
	// Group H
	"EGY": 0, "NGA": 1, "GHA": 2, "CMR": 3,
	// Group I
	"SWE": 0, "DEN": 1, "NOR": 2, "FIN": 3,
	// Group J
	"SUI": 0, "AUT": 1, "POL": 2, "UKR": 3,
	// Group K
	"TUR": 0, "GRE": 1, "CZE": 2, "SVK": 3,
	// Group L
	"WAL": 0, "SCO": 1, "IRL": 2, "NIR": 3,
}

func simulateGroupStageDeterministic(ctx context.Context, contestRepo *postgres.ContestRepository, db *sql.DB, contestID string) {
	// 1. Play Group Stage matches (Round = 0)
	rows, err := db.QueryContext(ctx, `
		SELECT m.id, c1.code, c2.code 
		FROM matches m
		JOIN countries c1 ON m.country1_id = c1.id
		JOIN countries c2 ON m.country2_id = c2.id
		WHERE m.contest_id = $1 AND m.round = 0
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query group matches: %v", err)
	}

	type matchInfo struct {
		id, c1, c2 string
	}
	var groupMatches []matchInfo
	for rows.Next() {
		var m matchInfo
		if err := rows.Scan(&m.id, &m.c1, &m.c2); err != nil {
			log.Fatalf("failed to scan group match: %v", err)
		}
		groupMatches = append(groupMatches, m)
	}
	rows.Close()

	for _, gm := range groupMatches {
		idx1 := countryIndexInGroup[gm.c1]
		idx2 := countryIndexInGroup[gm.c2]
		var g1, g2 int
		if idx1 < idx2 {
			g1 = 3 - idx1
			g2 = 0
		} else {
			g1 = 0
			g2 = 3 - idx2
		}

		match := entity.Match{
			Country1:      &entity.Country{Code: gm.c1},
			Country2:      &entity.Country{Code: gm.c2},
			Country1Goals: &g1,
			Country2Goals: &g2,
			Round:         0,
		}
		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to play group match: %v", err)
		}
	}

	// 2. Finalize Group Standings (A-L)
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	for _, letter := range letters {
		orderRows, err := db.QueryContext(ctx, `
			SELECT c.code 
			FROM group_standings gs
			JOIN countries c ON gs.country_id = c.id
			WHERE gs.contest_id = $1 AND gs.letter = $2
			ORDER BY gs.points DESC, gs.gd DESC, gs.gf DESC, c.code ASC
		`, contestID, letter)
		if err != nil {
			log.Fatalf("failed to query order for group %s: %v", letter, err)
		}

		var orderedCodes []string
		for orderRows.Next() {
			var code string
			if err := orderRows.Scan(&code); err != nil {
				log.Fatalf("failed to scan country code: %v", err)
			}
			orderedCodes = append(orderedCodes, code)
		}
		orderRows.Close()

		if err := contestRepo.FinalizeGroupRankings(ctx, contestID, letter, orderedCodes); err != nil {
			log.Fatalf("failed to finalize rankings for group %s: %v", letter, err)
		}
	}

	// 3. Finalize Third Place Qualifiers dynamically based on actual standings
	thirdRows, err := db.QueryContext(ctx, `
		SELECT gs.letter
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1 AND gs.rank = 3
		ORDER BY gs.points DESC, gs.gd DESC, gs.gf DESC, c.code ASC
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query third place standings: %v", err)
	}

	var sortedThirdPlaceLetters []string
	for thirdRows.Next() {
		var letter string
		if err := thirdRows.Scan(&letter); err != nil {
			log.Fatalf("failed to scan third place letter: %v", err)
		}
		sortedThirdPlaceLetters = append(sortedThirdPlaceLetters, letter)
	}
	thirdRows.Close()

	if len(sortedThirdPlaceLetters) != 12 {
		log.Fatalf("expected exactly 12 third-place teams, got %d", len(sortedThirdPlaceLetters))
	}

	qualifierMap := make(map[string]bool)
	for i, letter := range sortedThirdPlaceLetters {
		qualifierMap[letter] = (i < 8)
	}

	for _, letter := range letters {
		isQualifier := qualifierMap[letter]
		if err := contestRepo.FinalizeThirdPlaceQualifier(ctx, contestID, letter, isQualifier); err != nil {
			log.Fatalf("failed to finalize third place qualifier for group %s: %v", letter, err)
		}
	}

	// 4. Set up Round of 32 knockout matches (Round = 1) without goals or penalties
	advRows, err := db.QueryContext(ctx, `
		SELECT c.code 
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1 AND (gs.rank = 1 OR gs.rank = 2 OR (gs.rank = 3 AND gs.is_third_place_qualifier = true))
		ORDER BY gs.letter ASC, gs.rank ASC
	`, contestID)
	if err != nil {
		log.Fatalf("failed to query advancing teams: %v", err)
	}

	var advancingTeams []string
	for advRows.Next() {
		var code string
		if err := advRows.Scan(&code); err != nil {
			log.Fatalf("failed to scan advancing team code: %v", err)
		}
		advancingTeams = append(advancingTeams, code)
	}
	advRows.Close()

	if len(advancingTeams) != 32 {
		log.Fatalf("expected exactly 32 advancing teams, got %d", len(advancingTeams))
	}

	for i := 0; i < 16; i++ {
		c1 := advancingTeams[2*i]
		c2 := advancingTeams[2*i+1]

		matchIdx := i
		match := entity.Match{
			Round:      1,
			RoundIndex: &matchIdx,
			Country1:   &entity.Country{Code: c1},
			Country2:   &entity.Country{Code: c2},
		}

		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to update Round of 32 match index %d: %v", i, err)
		}
	}
}

func simulateKnockoutStageDeterministic(ctx context.Context, contestRepo *postgres.ContestRepository, db *sql.DB, contestID string) {
	// Play Round of 32 knockout matches (Round = 1) by adding goals and penalties (Country 1 wins)
	for i := 0; i < 16; i++ {
		g1 := 2
		g2 := 1
		matchIdx := i
		match := entity.Match{
			Round:             1,
			RoundIndex:        &matchIdx,
			Country1Goals:     &g1,
			Country2Goals:     &g2,
		}

		if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
			log.Fatalf("failed to play Round of 32 match index %d: %v", i, err)
		}
	}

	// Play subsequent rounds (Rounds 2 to 5)
	for round := 2; round <= 5; round++ {
		matchRows, err := db.QueryContext(ctx, `
			SELECT m.round_index, c1.code, c2.code 
			FROM matches m
			JOIN countries c1 ON m.country1_id = c1.id
			JOIN countries c2 ON m.country2_id = c2.id
			WHERE m.contest_id = $1 AND m.round = $2
			ORDER BY m.round_index ASC
		`, contestID, round)
		if err != nil {
			log.Fatalf("failed to query Round %d matches: %v", round, err)
		}

		type koMatch struct {
			idx    int
			c1, c2 string
		}
		var koMatches []koMatch
		for matchRows.Next() {
			var km koMatch
			if err := matchRows.Scan(&km.idx, &km.c1, &km.c2); err != nil {
				log.Fatalf("failed to scan Round %d match: %v", round, err)
			}
			koMatches = append(koMatches, km)
		}
		matchRows.Close()

		for _, km := range koMatches {
			g1 := 2
			g2 := 1
			mIdx := km.idx
			match := entity.Match{
				Round:             round,
				RoundIndex:        &mIdx,
				Country1Goals:     &g1,
				Country2Goals:     &g2,
			}

			if err := contestRepo.UpdateMatch(ctx, contestID, match); err != nil {
				log.Fatalf("failed to play Round %d match index %d: %v", round, km.idx, err)
			}
		}
	}
}

func seedGroupPicksForJoey(ctx context.Context, picksRepo *postgres.PicksRepository, db *sql.DB, userID, contestID string, countries []entity.Country, letters []string, isPerfect bool) error {
	rows, err := db.QueryContext(ctx, `
		SELECT c.code, gs.letter 
		FROM group_standings gs
		JOIN countries c ON gs.country_id = c.id
		WHERE gs.contest_id = $1
	`, contestID)
	if err != nil {
		return err
	}
	defer rows.Close()

	groupMap := make(map[string][]entity.Country)
	for rows.Next() {
		var code, letter string
		if err := rows.Scan(&code, &letter); err != nil {
			return err
		}
		var matchCountry entity.Country
		for _, c := range countries {
			if c.Code == code {
				matchCountry = c
				break
			}
		}
		groupMap[letter] = append(groupMap[letter], matchCountry)
	}

	var picks []entity.GroupPick
	for _, letter := range letters {
		groupCountries, ok := groupMap[letter]
		if !ok || len(groupCountries) == 0 {
			continue
		}

		sorted := make([]entity.Country, len(groupCountries))
		copy(sorted, groupCountries)
		// Sort countries by their pre-defined group index
		sort.Slice(sorted, func(i, j int) bool {
			return countryIndexInGroup[sorted[i].Code] < countryIndexInGroup[sorted[j].Code]
		})

		var entries []entity.GroupPickEntry
		for idx, t := range sorted {
			entries = append(entries, entity.GroupPickEntry{
				Country: t,
				Place:   idx + 1,
			})
		}

		isWildcardGroup := (letter == "A" || letter == "C" || letter == "D" || letter == "E" || letter == "F" || letter == "H" || letter == "K" || letter == "L")
		extraQual := isWildcardGroup
		if !isPerfect {
			extraQual = !isWildcardGroup
		}

		picks = append(picks, entity.GroupPick{
			Letter:         letter,
			Entries:        entries,
			ExtraQualifier: extraQual,
		})
	}

	return picksRepo.CreateGroupPicks(ctx, userID, contestID, picks)
}

func seedKnockoutPicksForJoey(ctx context.Context, picksRepo *postgres.PicksRepository, db *sql.DB, userID, contestID string, isPerfect bool) error {
	// 1. Query the 16 Round of 32 matches (Round = 1) from the database
	rows, err := db.QueryContext(ctx, `
		SELECT m.round_index, c1.code, c1.full_name, c2.code, c2.full_name
		FROM matches m
		JOIN countries c1 ON m.country1_id = c1.id
		JOIN countries c2 ON m.country2_id = c2.id
		WHERE m.contest_id = $1 AND m.round = 1
		ORDER BY m.round_index ASC
	`, contestID)
	if err != nil {
		return fmt.Errorf("failed to query Round of 32 matches: %w", err)
	}
	defer rows.Close()

	type r32Match struct {
		idx    int
		c1, c2 entity.Country
	}
	var r32Matches []r32Match
	for rows.Next() {
		var rm r32Match
		if err := rows.Scan(&rm.idx, &rm.c1.Code, &rm.c1.FullName, &rm.c2.Code, &rm.c2.FullName); err != nil {
			return fmt.Errorf("failed to scan Round of 32 match: %w", err)
		}
		r32Matches = append(r32Matches, rm)
	}

	if len(r32Matches) != 16 {
		return fmt.Errorf("expected 16 Round of 32 matches, found %d", len(r32Matches))
	}

	// 2. Select winners of Round of 32 (Round 1) to advance to Round of 16 (Round 2)
	var entries []entity.KnockoutPickEntry
	var r16Picks []entity.Country
	for _, m := range r32Matches {
		winner := m.c1
		r16Picks = append(r16Picks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 2})
	}

	// 3. Select winners of Round of 16 (Round 2) to advance to Quarterfinals (Round 3)
	var qfPicks []entity.Country
	for i := 0; i < 8; i++ {
		winner := r16Picks[i*2]
		qfPicks = append(qfPicks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 3})
	}

	// 4. Select winners of Quarterfinals (Round 3) to advance to Semifinals (Round 4)
	var sfPicks []entity.Country
	for i := 0; i < 4; i++ {
		winner := qfPicks[i*2]
		sfPicks = append(sfPicks, winner)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 4})
	}

	// 5. Select winners of Semifinals (Round 4) to advance to Finals (Round 5)
	var finalPicks []entity.Country
	var sfLosers []entity.Country
	for i := 0; i < 2; i++ {
		winner := sfPicks[i*2]
		loser := sfPicks[i*2+1]
		finalPicks = append(finalPicks, winner)
		sfLosers = append(sfLosers, loser)
		entries = append(entries, entity.KnockoutPickEntry{Country: winner, Round: 5})
	}

	// 6. Select Champion (Round 6) from the Finals
	champion := finalPicks[0]
	entries = append(entries, entity.KnockoutPickEntry{Country: champion, Round: 6})

	// 7. Select Third Place Winner (Round 7) from the Semifinals Losers
	var thirdPlaceWinner entity.Country
	if isPerfect {
		thirdPlaceWinner = sfLosers[0]
	} else {
		thirdPlaceWinner = sfLosers[1]
	}
	entries = append(entries, entity.KnockoutPickEntry{Country: thirdPlaceWinner, Round: 7})

	pick := entity.KnockoutPick{Entries: entries}
	return picksRepo.CreateKnockoutPicks(ctx, userID, contestID, pick)
}
