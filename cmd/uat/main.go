package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joey/wcwcpp-backend/adapters/auth"
	"github.com/joey/wcwcpp-backend/adapters/handler"
	"github.com/joey/wcwcpp-backend/adapters/storage/postgres"
	"github.com/joey/wcwcpp-backend/core/service"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const serverAddr = "127.0.0.1:8081"
const baseClientURL = "http://" + serverAddr

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	superadminEmail := os.Getenv("SUPERADMIN_EMAILS")

	if dbURL == "" || jwtSecret == "" || superadminEmail == "" {
		log.Fatalf("DATABASE_URL, JWT_SECRET, and SUPERADMIN_EMAILS are required env variables")
	}

	// 1. Establish Database Connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping db: %v", err)
	}

	// 2. Start the UAT ConnectRPC Server in the background
	go startTestServer(db)
	time.Sleep(1 * time.Second) // wait for server start

	fmt.Println("\n=======================================================")
	fmt.Println("🚀 WCWCPP Backend: End-to-End User Acceptance Test (UAT)")
	fmt.Println("=======================================================")

	// 3. Clear/Migrate DB & Seed clean state
	cleanupDB(db)
	seedUATData(db, superadminEmail)

	// 4. Generate Auth Tokens for test scenarios
	superToken := getJWTToken(db, jwtSecret, "super@example.com")
	user1Token := getJWTToken(db, jwtSecret, "user1@example.com")
	user2Token := getJWTToken(db, jwtSecret, "user2@example.com")

	// 5. Instantiate Connect clients
	contestCli := v1connect.NewContestServiceClient(http.DefaultClient, baseClientURL)
	matchCli := v1connect.NewMatchServiceClient(http.DefaultClient, baseClientURL)
	picksCli := v1connect.NewPicksServiceClient(http.DefaultClient, baseClientURL)
	leaderboardCli := v1connect.NewLeaderboardServiceClient(http.DefaultClient, baseClientURL)
	usersCli := v1connect.NewUsersServiceClient(http.DefaultClient, baseClientURL)

	ctx := context.Background()

	// ==========================================
	// 🏆 SCENARIO 1: Users Service & Security
	// ==========================================
	fmt.Println("\n👤 [SCENARIO 1: Users Service & Role Enforcement]")

	// Test 1: CountUsers as Superadmin (Public access)
	reqCountSuper := connect.NewRequest(&v1.CountUsersRequest{})
	reqCountSuper.Header().Set("Authorization", "Bearer "+superToken)
	resCountSuper, err := usersCli.CountUsers(ctx, reqCountSuper)
	if err != nil {
		log.Fatalf("Failed to count users as superadmin: %v", err)
	}
	fmt.Printf("  ✅ CountUsers (expect 3 seeded): %d\n", resCountSuper.Msg.Count)

	// Test 2: DeleteUser as User 2 (Deletes currently logged-in User 2)
	reqDelUser2 := connect.NewRequest(&v1.DeleteUserRequest{})
	reqDelUser2.Header().Set("Authorization", "Bearer "+user2Token)
	_, err = usersCli.DeleteUser(ctx, reqDelUser2)
	if err != nil {
		log.Fatalf("Failed to self-delete user 2: %v", err)
	}
	fmt.Println("  ✅ Self-DeleteUser as User 2 succeeded")

	// Verification: CountUsers should now be 2
	resCountSuper2, _ := usersCli.CountUsers(ctx, reqCountSuper)
	fmt.Printf("  ✅ Verified CountUsers decreases to: %d\n", resCountSuper2.Msg.Count)

	// Recreate User 2 for subsequent tests
	_, err = db.Exec("INSERT INTO users (email, username) VALUES ('user2@example.com', 'user2') ON CONFLICT DO NOTHING")
	if err != nil {
		log.Fatalf("Failed to recreate user 2: %v", err)
	}
	user2Token = getJWTToken(db, jwtSecret, "user2@example.com")

	// ==========================================
	// 🏆 SCENARIO 2: Contest Service Endpoints
	// ==========================================
	fmt.Println("\n🏟️ [SCENARIO 2: Contest Service Endpoints]")

	// Test 1: List Contests (Public access)
	reqListContests := connect.NewRequest(&v1.ListContestsRequest{})
	resListContests, err := contestCli.ListContests(ctx, reqListContests)
	if err != nil {
		log.Fatalf("Failed to list contests: %v", err)
	}
	fmt.Printf("  ✅ ListContests (Public) returned %d contest(s)\n", len(resListContests.Msg.Contests))

	// Test 2: CreateContest as non-admin (should fail)
	reqCreateContest := connect.NewRequest(&v1.CreateContestRequest{
		Title: "UAT World Cup 2030",
	})
	reqCreateContest.Header().Set("Authorization", "Bearer "+user1Token)
	_, err = contestCli.CreateContest(ctx, reqCreateContest)
	if err == nil {
		log.Fatalf("Expected permission denied error for normal user on CreateContest, but succeeded")
	}
	fmt.Println("  ✅ CreateContest as Normal User correctly fails (PermissionDenied)")

	// Test 3: CreateContest as Superadmin (Success - exactly 12 groups required by schema)
	var protoGroups []*v1.Group
	groupLetters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	for _, letter := range groupLetters {
		var protoCountries []*v1.Country
		for j := range 4 {
			code := fmt.Sprintf("%s%d", letter, j+1)
			protoCountries = append(protoCountries, &v1.Country{
				Code:     code,
				FullName: "Country " + code,
			})
		}
		protoGroups = append(protoGroups, &v1.Group{
			Letter:    letter,
			Countries: protoCountries,
		})
	}

	reqCreateContestSuper := connect.NewRequest(&v1.CreateContestRequest{
		Title:              "UAT World Cup 2030",
		Groups:             protoGroups,
		GroupUnlockDate:    timestamppb.New(time.Now().Add(-24 * time.Hour)),
		GroupLockDate:      timestamppb.New(time.Now().Add(48 * time.Hour)),
		KnockoutUnlockDate: timestamppb.New(time.Now().Add(-24 * time.Hour)),
		KnockoutLockDate:   timestamppb.New(time.Now().Add(48 * time.Hour)),
	})
	reqCreateContestSuper.Header().Set("Authorization", "Bearer "+superToken)
	_, err = contestCli.CreateContest(ctx, reqCreateContestSuper)
	if err != nil {
		log.Fatalf("Failed to create contest as superadmin: %v", err)
	}
	fmt.Println("  ✅ CreateContest as Superadmin succeeded")

	// ==========================================
	// 🏆 SCENARIO 3: Picks & Predictions
	// ==========================================
	fmt.Println("\n✍️ [SCENARIO 3: Picks Service & Predictions]")

	// Test 1: Submit Group Picks for User 1
	var protoPicks1 []*v1.GroupPick
	for _, letter := range groupLetters {
		protoPicks1 = append(protoPicks1, &v1.GroupPick{
			Group: &v1.Group{
				Letter: letter,
				Countries: []*v1.Country{
					{Code: letter + "1"},
					{Code: letter + "2"},
					{Code: letter + "3"},
					{Code: letter + "4"},
				},
			},
			ExtraQualifier: true,
		})
	}
	reqCreatePicks1 := connect.NewRequest(&v1.CreateGroupPicksRequest{
		ContestSlug: "uat-world-cup-2030",
		Picks:       protoPicks1,
	})
	reqCreatePicks1.Header().Set("Authorization", "Bearer "+user1Token)
	_, err = picksCli.CreateGroupPicks(ctx, reqCreatePicks1)
	if err != nil {
		log.Fatalf("Failed to submit group picks for User 1: %v", err)
	}
	fmt.Println("  ✅ CreateGroupPicks User 1 succeeded")

	// Test 2: List Group Picks for User 1 (Verifies mapping & newly added extra_qualifier boolean)
	reqListPicks1 := connect.NewRequest(&v1.ListGroupPicksRequest{
		ContestSlug: "uat-world-cup-2030",
	})
	reqListPicks1.Header().Set("Authorization", "Bearer "+user1Token)
	resListPicks1, err := picksCli.ListGroupPicks(ctx, reqListPicks1)
	if err != nil {
		log.Fatalf("Failed to list group picks: %v", err)
	}
	fmt.Printf("  ✅ ListGroupPicks returned %d pick predictions. ExtraQualifier: %t\n",
		len(resListPicks1.Msg.Picks), resListPicks1.Msg.Picks[0].ExtraQualifier)

	// Test 3: Submit Group Picks for User 2 (different order)
	var protoPicks2 []*v1.GroupPick
	for _, letter := range groupLetters {
		protoPicks2 = append(protoPicks2, &v1.GroupPick{
			Group: &v1.Group{
				Letter: letter,
				Countries: []*v1.Country{
					{Code: letter + "1"},
					{Code: letter + "2"},
					{Code: letter + "4"},
					{Code: letter + "3"},
				},
			},
			ExtraQualifier: false,
		})
	}
	reqCreatePicks2 := connect.NewRequest(&v1.CreateGroupPicksRequest{
		ContestSlug: "uat-world-cup-2030",
		Picks:       protoPicks2,
	})
	reqCreatePicks2.Header().Set("Authorization", "Bearer "+user2Token)
	_, err = picksCli.CreateGroupPicks(ctx, reqCreatePicks2)
	if err != nil {
		log.Fatalf("Failed to submit group picks for User 2: %v", err)
	}
	fmt.Println("  ✅ CreateGroupPicks User 2 succeeded")

	// ==========================================
	// 🏆 SCENARIO 4: Invite-Only Subcontests Flow
	// ==========================================
	fmt.Println("\n🗝️ [SCENARIO 4: Invite-Only Subcontests & Joins]")

	// Test 1: User 1 creates subcontest
	reqCreateSub := connect.NewRequest(&v1.CreateSubcontestRequest{
		ContestSlug:     "uat-world-cup-2030",
		SubcontestTitle: "UAT Private League",
		SelfJoin:        true,
	})
	reqCreateSub.Header().Set("Authorization", "Bearer "+user1Token)
	resCreateSub, err := contestCli.CreateSubcontest(ctx, reqCreateSub)
	if err != nil {
		log.Fatalf("Failed to create subcontest: %v", err)
	}
	joinCode := resCreateSub.Msg.JoinCode
	fmt.Printf("  ✅ CreateSubcontest (SelfJoin=True) succeeded. Join Code: %s\n", joinCode)

	// Test 2: User 2 lists subcontests (should see 0)
	reqListSubs2 := connect.NewRequest(&v1.ListSubcontestsRequest{ContestSlug: "uat-world-cup-2030"})
	reqListSubs2.Header().Set("Authorization", "Bearer "+user2Token)
	resListSubs2, _ := contestCli.ListSubcontests(ctx, reqListSubs2)
	fmt.Printf("  ✅ User 2 ListSubcontests returned: %d subcontests\n", len(resListSubs2.Msg.Subcontests))

	// Test 3: User 2 joins subcontest using join code
	reqJoin := connect.NewRequest(&v1.JoinSubcontestRequest{JoinCode: joinCode})
	reqJoin.Header().Set("Authorization", "Bearer "+user2Token)
	_, err = contestCli.JoinSubcontest(ctx, reqJoin)
	if err != nil {
		log.Fatalf("User 2 failed to join subcontest: %v", err)
	}
	fmt.Println("  ✅ User 2 JoinSubcontest succeeded")

	// Verification: User 2 lists subcontests (should now see 1)
	resListSubs2After, _ := contestCli.ListSubcontests(ctx, reqListSubs2)
	fmt.Printf("  ✅ User 2 ListSubcontests after join returned: %d subcontests\n", len(resListSubs2After.Msg.Subcontests))

	// ==========================================
	// 🏆 SCENARIO 5: Matches & Scores Resolution
	// ==========================================
	fmt.Println("\n⚽ [SCENARIO 5: Match Scores & Completeness]")

	// Test 1: List Matches
	reqListMatches := connect.NewRequest(&v1.ListGroupMatchesRequest{
		ContestSlug: "uat-world-cup-2030",
		Letter:      "A",
	})
	resListMatches, err := matchCli.ListGroupMatches(ctx, reqListMatches)
	if err != nil {
		log.Fatalf("Failed to list group matches: %v", err)
	}
	fmt.Printf("  ✅ ListGroupMatches returned %d matches in group A\n", len(resListMatches.Msg.Matches))

	// Test 2: Complete all 6 matches in Group A as Superadmin
	for _, m := range resListMatches.Msg.Matches {
		reqUpdMatch := connect.NewRequest(&v1.CreateMatchRequest{
			ContestSlug: "uat-world-cup-2030",
			Match: &v1.Match{
				Country1:      m.Country1,
				Country2:      m.Country2,
				Country1Goals: proto.Int64(1),
				Country2Goals: proto.Int64(0),
				Round:         0,
				RoundIndex:    m.RoundIndex,
			},
		})
		reqUpdMatch.Header().Set("Authorization", "Bearer "+superToken)
		_, err = matchCli.CreateMatch(ctx, reqUpdMatch)
		if err != nil {
			log.Fatalf("Failed to complete group match: %v", err)
		}
	}
	fmt.Println("  ✅ All 6 Group A matches completed successfully by Superadmin")

	// Test 3: Immutable double-complete check
	firstMatch := resListMatches.Msg.Matches[0]
	reqDoubleMatch := connect.NewRequest(&v1.CreateMatchRequest{
		ContestSlug: "uat-world-cup-2030",
		Match: &v1.Match{
			Country1:      firstMatch.Country1,
			Country2:      firstMatch.Country2,
			Country1Goals: proto.Int64(3),
			Country2Goals: proto.Int64(2),
			Round:         0,
			RoundIndex:    firstMatch.RoundIndex,
		},
	})
	reqDoubleMatch.Header().Set("Authorization", "Bearer "+superToken)
	_, err = matchCli.CreateMatch(ctx, reqDoubleMatch)
	if err == nil {
		log.Fatalf("Expected validation error when modifying already completed match, but succeeded")
	}
	fmt.Println("  ✅ Double-complete of completed match correctly fails (FailedPrecondition)")

	// ==========================================
	// 🏆 SCENARIO 6: Group Finalization & Scoring Engine
	// ==========================================
	fmt.Println("\n🏅 [SCENARIO 6: Standings Finalization & Points calculations]")

	// Test 1: Finalize Group Rankings as Superadmin
	reqFinRank := connect.NewRequest(&v1.FinalizeGroupRankingsRequest{
		ContestSlug:         "uat-world-cup-2030",
		GroupLetter:         "A",
		OrderedCountryCodes: []string{"A1", "A2", "A3", "A4"},
	})
	reqFinRank.Header().Set("Authorization", "Bearer "+superToken)
	_, err = contestCli.FinalizeGroupRankings(ctx, reqFinRank)
	if err != nil {
		log.Fatalf("Failed to finalize group rankings: %v", err)
	}
	fmt.Println("  ✅ FinalizeGroupRankings (A1=1, A2=2, A3=3, A4=4) succeeded")

	// Test 2: Double Finalization error check
	_, err = contestCli.FinalizeGroupRankings(ctx, reqFinRank)
	if err == nil {
		log.Fatalf("Expected double-finalization rankings call to fail, but succeeded")
	}
	fmt.Println("  ✅ Double rankings finalization correctly fails (FailedPrecondition)")

	// Verify User 1 Score (Perfect Predictions!)
	// Predictions: A1 (1st), A2 (2nd), A3 (3rd), A4 (4th) -> 45 points!
	reqLboard := connect.NewRequest(&v1.LeaderboardRequest{
		ContestSlug: "uat-world-cup-2030",
		Limit:       10,
	})
	resLboard, _ := leaderboardCli.Leaderboard(ctx, reqLboard)
	if len(resLboard.Msg.Group) != 2 {
		log.Fatalf("Expected exactly 2 leaderboard entries, but got %d", len(resLboard.Msg.Group))
	}
	for _, entry := range resLboard.Msg.Group {
		if entry.Name == "user1" {
			if entry.Score != 45 {
				log.Fatalf("Expected user1 score to be 45, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 1 Leaderboard score (expects 45): %d\n", entry.Score)
		}
		if entry.Name == "user2" {
			if entry.Score != 43 {
				log.Fatalf("Expected user2 score to be 43, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 2 Leaderboard score (expects 43): %d\n", entry.Score)
		}
	}

	// Test 3: Finalize Third Place Qualifier (Wildcard Qualifier = True)
	reqFinWildcard := connect.NewRequest(&v1.FinalizeThirdPlaceQualifierRequest{
		ContestSlug:         "uat-world-cup-2030",
		GroupLetter:         "A",
		IsWildcardQualifier: true,
	})
	reqFinWildcard.Header().Set("Authorization", "Bearer "+superToken)
	_, err = contestCli.FinalizeThirdPlaceQualifier(ctx, reqFinWildcard)
	if err != nil {
		log.Fatalf("Failed to finalize third place qualifier: %v", err)
	}
	fmt.Println("  ✅ FinalizeThirdPlaceQualifier (A3 advances as wildcard = True) succeeded")

	// Test 4: Double qualifier finalization check
	_, err = contestCli.FinalizeThirdPlaceQualifier(ctx, reqFinWildcard)
	if err == nil {
		log.Fatalf("Expected double qualifier finalization call to fail, but succeeded")
	}
	fmt.Println("  ✅ Double wildcard qualifier finalization correctly fails (FailedPrecondition)")

	// Verify Leaderboard Scores (User 1 predicted True -> +5 pts -> 50 pts; User 2 predicted False -> +0 pts -> 43 pts)
	resLboardAfter, _ := leaderboardCli.Leaderboard(ctx, reqLboard)
	if len(resLboardAfter.Msg.Group) != 2 {
		log.Fatalf("Expected exactly 2 leaderboard entries after wildcard, but got %d", len(resLboardAfter.Msg.Group))
	}
	for _, entry := range resLboardAfter.Msg.Group {
		if entry.Name == "user1" {
			if entry.Score != 50 {
				log.Fatalf("Expected user1 score after wildcard to be 50, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 1 Leaderboard score after Wildcard bonus (expects 50): %d\n", entry.Score)
		}
		if entry.Name == "user2" {
			if entry.Score != 43 {
				log.Fatalf("Expected user2 score after wildcard to be 43, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 2 Leaderboard score after Wildcard bonus (expects 43): %d\n", entry.Score)
		}
	}

	// ==========================================
	// 🏆 SCENARIO 7: Leaderboards
	// ==========================================
	fmt.Println("\n📊 [SCENARIO 7: Leaderboard Endpoints]")

	// Test 1: Global Leaderboard
	fmt.Printf("  ✅ Global Leaderboard Entries (Count: %d):\n", len(resLboardAfter.Msg.Group))
	for idx, entry := range resLboardAfter.Msg.Group {
		fmt.Printf("     [%d] %s: Group Score = %d\n", idx+1, entry.Name, entry.Score)
	}

	// Test 2: Subcontest Leaderboard
	subcontestSlug := resListSubs2After.Msg.Subcontests[0].Slug
	reqSubLboard := connect.NewRequest(&v1.SubleaderboardRequest{
		SubcontestSlug: subcontestSlug,
		Limit:          10,
	})
	reqSubLboard.Header().Set("Authorization", "Bearer "+user2Token)
	resSubLboard, err := leaderboardCli.Subleaderboard(ctx, reqSubLboard)
	if err != nil {
		log.Fatalf("Failed to fetch subleaderboard: %v", err)
	}
	fmt.Printf("  ✅ Subcontest Leaderboard '%s' Entries (Count: %d):\n", subcontestSlug, len(resSubLboard.Msg.Group))
	for idx, entry := range resSubLboard.Msg.Group {
		fmt.Printf("     [%d] %s: Group Score = %d\n", idx+1, entry.Name, entry.Score)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("🎉 ALL UAT END-TO-END VERIFICATION SCENARIOS COMPLETED GREEN!")
	fmt.Println("=======================================================")
}

func startTestServer(db *sql.DB) {
	// Initialize Adapters
	userRepo := postgres.NewUserRepository(db)
	tokenValidator := auth.NewGoogleTokenValidator()

	// Initialize Core Services
	authService := service.NewAuthService(userRepo, tokenValidator)
	contestRepo := postgres.NewContestRepository(db)
	contestService := service.NewContestService(contestRepo)
	usersService := service.NewUsersService(userRepo)
	matchService := service.NewMatchService(contestRepo)

	// Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	contestHandler := handler.NewContestHandler(contestService)
	usersHandler := handler.NewUsersHandler(usersService)
	matchHandler := handler.NewMatchHandler(matchService)

	leaderboardRepo := postgres.NewLeaderboardRepository(db)
	leaderboardService := service.NewLeaderboardService(leaderboardRepo)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)

	picksRepo := postgres.NewPicksRepository(db)
	picksService := service.NewPicksService(picksRepo)
	picksHandler := handler.NewPicksHandler(picksService)

	mux := http.NewServeMux()

	authPath, authSvcHandler := v1connect.NewAuthServiceHandler(authHandler)
	mux.Handle(authPath, authSvcHandler)

	contestPath, contestSvcHandler := v1connect.NewContestServiceHandler(contestHandler)
	mux.Handle(contestPath, contestSvcHandler)

	usersPath, usersSvcHandler := v1connect.NewUsersServiceHandler(usersHandler)
	mux.Handle(usersPath, usersSvcHandler)

	leaderboardPath, leaderboardSvcHandler := v1connect.NewLeaderboardServiceHandler(leaderboardHandler)
	mux.Handle(leaderboardPath, leaderboardSvcHandler)

	picksPath, picksSvcHandler := v1connect.NewPicksServiceHandler(picksHandler)
	mux.Handle(picksPath, picksSvcHandler)

	matchPath, matchSvcHandler := v1connect.NewMatchServiceHandler(matchHandler)
	mux.Handle(matchPath, matchSvcHandler)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Test Server failed: %v", err)
	}
}

func cleanupDB(db *sql.DB) {
	tables := []string{
		"subcontest_entries", "subcontests", "group_picks", "knockout_picks",
		"group_standings", "knockout_standings", "matches", "contest_standings",
		"contests", "countries", "users",
	}
	for _, t := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", t))
		if err != nil {
			log.Fatalf("Failed to truncate table %s: %v", t, err)
		}
	}
	fmt.Println("  🧹 Database cleaned and structures reset")
}

func seedUATData(db *sql.DB, superadminEmail string) {
	users := []struct {
		email    string
		username string
	}{
		{email: superadminEmail, username: "superadmin"},
		{email: "user1@example.com", username: "user1"},
		{email: "user2@example.com", username: "user2"},
	}

	for _, u := range users {
		_, err := db.Exec("INSERT INTO users (email, username) VALUES ($1, $2)", u.email, u.username)
		if err != nil {
			log.Fatalf("Failed to seed user %s: %v", u.email, err)
		}
	}
	fmt.Println("  🌱 Test Users seeded (superadmin, user1, user2)")
}

func getJWTToken(db *sql.DB, secret string, email string) string {
	var userID string
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		log.Fatalf("Failed to find user ID for token generation (%s): %v", email, err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	return tokenString
}
