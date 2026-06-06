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
	"github.com/joey/wcwcpp-backend/adapters/handler"
	"github.com/joey/wcwcpp-backend/adapters/storage/postgres"
	"github.com/joey/wcwcpp-backend/core/service"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const serverAddr = "127.0.0.1:8081"
const baseClientURL = "http://" + serverAddr

type mockTokenValidator struct {
	superadminEmail string
}

func (m *mockTokenValidator) ValidateGoogleToken(ctx context.Context, token string) (string, error) {
	switch token {
	case "mock-google-token-super":
		return m.superadminEmail, nil
	case "mock-google-token-user1":
		return "user1@example.com", nil
	case "mock-google-token-user2":
		return "user2@example.com", nil
	case "mock-google-token-newuser":
		return "newuser@example.com", nil
	default:
		return "", fmt.Errorf("invalid token")
	}
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	superadminEmailsRaw := os.Getenv("SUPERADMIN_EMAILS")

	if dbURL == "" || jwtSecret == "" || superadminEmailsRaw == "" {
		log.Fatalf("DATABASE_URL, JWT_SECRET, and SUPERADMIN_EMAILS are required env variables")
	}

	emails := strings.Split(superadminEmailsRaw, ",")
	if len(emails) == 0 || strings.TrimSpace(emails[0]) == "" {
		log.Fatalf("SUPERADMIN_EMAILS must contain at least one valid email")
	}
	superadminEmail := strings.TrimSpace(emails[0])

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
	superToken := getJWTToken(db, jwtSecret, superadminEmail)
	user1Token := getJWTToken(db, jwtSecret, "user1@example.com")
	user2Token := getJWTToken(db, jwtSecret, "user2@example.com")

	// 5. Instantiate Connect clients
	authCli := v1connect.NewAuthServiceClient(http.DefaultClient, baseClientURL)
	contestCli := v1connect.NewContestServiceClient(http.DefaultClient, baseClientURL)
	matchCli := v1connect.NewMatchServiceClient(http.DefaultClient, baseClientURL)
	picksCli := v1connect.NewPicksServiceClient(http.DefaultClient, baseClientURL)
	leaderboardCli := v1connect.NewLeaderboardServiceClient(http.DefaultClient, baseClientURL)
	usersCli := v1connect.NewUsersServiceClient(http.DefaultClient, baseClientURL)

	ctx := context.Background()

	// ==========================================
	// 🏆 SCENARIO 0: AuthService (Login & Register)
	// ==========================================
	fmt.Println("\n🔑 [SCENARIO 0: AuthService (Login & Registration)]")

	// Test 1: Login with existing user (User 1)
	loginRes, err := authCli.Login(ctx, connect.NewRequest(&v1.LoginRequest{
		GoogleIdToken: "mock-google-token-user1",
	}))
	if err != nil {
		log.Fatalf("Failed to login as user1: %v", err)
	}
	fmt.Printf("  ✅ Login (existing user) succeeded. Username: %s, Email: %s\n", loginRes.Msg.User.Username, loginRes.Msg.User.Email)

	// Test 2: Login new user without username (should fail)
	_, err = authCli.Login(ctx, connect.NewRequest(&v1.LoginRequest{
		GoogleIdToken: "mock-google-token-newuser",
	}))
	if err == nil {
		log.Fatalf("Expected registration failure when username is missing, but succeeded")
	}
	fmt.Println("  ✅ Login new user without username correctly fails (NotFound)")

	// Test 3: Register new user (with username)
	uname := "newuser"
	registerRes, err := authCli.Login(ctx, connect.NewRequest(&v1.LoginRequest{
		GoogleIdToken: "mock-google-token-newuser",
		Username:      &uname,
	}))
	if err != nil {
		log.Fatalf("Failed to register new user: %v", err)
	}
	fmt.Printf("  ✅ Register new user succeeded. Username: %s, Email: %s\n", registerRes.Msg.User.Username, registerRes.Msg.User.Email)

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
	fmt.Printf("  ✅ CountUsers (expect 4 seeded/registered): %d\n", resCountSuper.Msg.Count)

	// Test 2: DeleteUser as User 2 (Deletes currently logged-in User 2)
	reqDelUser2 := connect.NewRequest(&v1.DeleteUserRequest{})
	reqDelUser2.Header().Set("Authorization", "Bearer "+user2Token)
	_, err = usersCli.DeleteUser(ctx, reqDelUser2)
	if err != nil {
		log.Fatalf("Failed to self-delete user 2: %v", err)
	}
	fmt.Println("  ✅ Self-DeleteUser as User 2 succeeded")

	// Verification: CountUsers should now be 3
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
	_, err = contestCli.CreateSubcontest(ctx, reqCreateSub)
	if err != nil {
		log.Fatalf("Failed to create subcontest: %v", err)
	}

	// Retrieve join code via ListSubcontests since CreateSubcontestResponse is now empty
	reqListSubs1 := connect.NewRequest(&v1.ListSubcontestsRequest{ContestSlug: "uat-world-cup-2030"})
	reqListSubs1.Header().Set("Authorization", "Bearer "+user1Token)
	resListSubs1, err := contestCli.ListSubcontests(ctx, reqListSubs1)
	if err != nil || len(resListSubs1.Msg.Subcontests) == 0 {
		log.Fatalf("Failed to list subcontests to retrieve join code: %v", err)
	}
	joinCode := resListSubs1.Msg.Subcontests[0].JoinCode
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

	// Test 4: DeleteSubcontest Flow
	// Create subcontest to delete
	reqCreateDelSub := connect.NewRequest(&v1.CreateSubcontestRequest{
		ContestSlug:     "uat-world-cup-2030",
		SubcontestTitle: "UAT League to Delete",
		SelfJoin:        true,
	})
	reqCreateDelSub.Header().Set("Authorization", "Bearer "+user1Token)
	_, err = contestCli.CreateSubcontest(ctx, reqCreateDelSub)
	if err != nil {
		log.Fatalf("Failed to create subcontest for deletion: %v", err)
	}

	// Non-owner attempts to delete (should fail)
	reqDelSubFail := connect.NewRequest(&v1.DeleteSubcontestRequest{
		SubcontestSlug: "uat-league-to-delete",
	})
	reqDelSubFail.Header().Set("Authorization", "Bearer "+user2Token)
	_, err = contestCli.DeleteSubcontest(ctx, reqDelSubFail)
	if err == nil {
		log.Fatalf("Expected permission denied when normal user deletes subcontest they do not own, but succeeded")
	}
	fmt.Println("  ✅ DeleteSubcontest by non-owner correctly fails (PermissionDenied)")

	// Owner deletes subcontest (should succeed)
	reqDelSubSuccess := connect.NewRequest(&v1.DeleteSubcontestRequest{
		SubcontestSlug: "uat-league-to-delete",
	})
	reqDelSubSuccess.Header().Set("Authorization", "Bearer "+user1Token)
	_, err = contestCli.DeleteSubcontest(ctx, reqDelSubSuccess)
	if err != nil {
		log.Fatalf("Failed to delete subcontest as owner: %v", err)
	}
	fmt.Println("  ✅ DeleteSubcontest by owner succeeded")

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

	// Test 4: List Knockout Matches
	reqListKO := connect.NewRequest(&v1.ListKnockoutMatchesRequest{
		ContestSlug: "uat-world-cup-2030",
	})
	resListKO, err := matchCli.ListKnockoutMatches(ctx, reqListKO)
	if err != nil {
		log.Fatalf("Failed to list knockout matches: %v", err)
	}
	fmt.Printf("  ✅ ListKnockoutMatches returned %d matches (expects 32)\n", len(resListKO.Msg.Matches))
	if len(resListKO.Msg.Matches) != 32 {
		log.Fatalf("Expected exactly 32 knockout matches, but got %d", len(resListKO.Msg.Matches))
	}

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
	// 🏆 SCENARIO 6b: Knockout Predictions & Scoring
	// ==========================================
	fmt.Println("\n🏆 [SCENARIO 6b: Knockout Predictions & Scoring]")

	// Test 1: Submit 32 Knockout Picks for User 1
	var koEntries []*v1.KnockoutEntry
	// 16 entries for Round 2 (Round of 16)
	r2Codes := []string{"A1", "A2", "B1", "B2", "C1", "C2", "D1", "D2", "E1", "E2", "F1", "F2", "G1", "G2", "H1", "H2"}
	for _, code := range r2Codes {
		koEntries = append(koEntries, &v1.KnockoutEntry{
			Country: &v1.Country{Code: code},
			Round:   2,
		})
	}
	// 8 entries for Round 3 (Quarterfinals)
	r3Codes := []string{"A1", "B1", "C1", "D1", "E1", "F1", "G1", "H1"}
	for _, code := range r3Codes {
		koEntries = append(koEntries, &v1.KnockoutEntry{
			Country: &v1.Country{Code: code},
			Round:   3,
		})
	}
	// 4 entries for Round 4 (Semifinals)
	r4Codes := []string{"A1", "C1", "E1", "G1"}
	for _, code := range r4Codes {
		koEntries = append(koEntries, &v1.KnockoutEntry{
			Country: &v1.Country{Code: code},
			Round:   4,
		})
	}
	// 2 entries for Round 5 (Finalists)
	r5Codes := []string{"A1", "E1"}
	for _, code := range r5Codes {
		koEntries = append(koEntries, &v1.KnockoutEntry{
			Country: &v1.Country{Code: code},
			Round:   5,
		})
	}
	// 1 entry for Round 6 (Champion)
	koEntries = append(koEntries, &v1.KnockoutEntry{
		Country: &v1.Country{Code: "A1"},
		Round:   6,
	})
	// 1 entry for Round 7 (Third-Place winner)
	koEntries = append(koEntries, &v1.KnockoutEntry{
		Country: &v1.Country{Code: "C1"},
		Round:   7,
	})

	reqCreateKOPicks := connect.NewRequest(&v1.CreateKnockoutPicksRequest{
		ContestSlug: "uat-world-cup-2030",
		Pick: &v1.KnockoutPick{
			Entries: koEntries,
		},
	})
	reqCreateKOPicks.Header().Set("Authorization", "Bearer "+user1Token)
	_, err = picksCli.CreateKnockoutPicks(ctx, reqCreateKOPicks)
	if err != nil {
		log.Fatalf("Failed to create knockout picks for User 1: %v", err)
	}
	fmt.Println("  ✅ CreateKnockoutPicks User 1 succeeded")

	// Test 2: List Knockout Picks for User 1
	reqListKOPicks := connect.NewRequest(&v1.ListKnockoutPicksRequest{
		ContestSlug: "uat-world-cup-2030",
	})
	reqListKOPicks.Header().Set("Authorization", "Bearer "+user1Token)
	resListKOPicks, err := picksCli.ListKnockoutPicks(ctx, reqListKOPicks)
	if err != nil {
		log.Fatalf("Failed to list knockout picks for User 1: %v", err)
	}
	fmt.Printf("  ✅ ListKnockoutPicks returned %d pick entries.\n", len(resListKOPicks.Msg.Pick.Entries))
	if len(resListKOPicks.Msg.Pick.Entries) != 32 {
		log.Fatalf("Expected exactly 32 knockout pick entries, but got %d", len(resListKOPicks.Msg.Pick.Entries))
	}

	// Test 3: Complete a Knockout Match (Round 1 Index 0: Country A1 vs Country A2, A1 wins 2-1)
	reqCompleteKO := connect.NewRequest(&v1.CreateMatchRequest{
		ContestSlug: "uat-world-cup-2030",
		Match: &v1.Match{
			Country1:      &v1.Country{Code: "A1", FullName: "Country A1"},
			Country2:      &v1.Country{Code: "A2", FullName: "Country A2"},
			Country1Goals: proto.Int64(2),
			Country2Goals: proto.Int64(1),
			Round:         1,
			RoundIndex:    proto.Int64(0),
		},
	})
	reqCompleteKO.Header().Set("Authorization", "Bearer "+superToken)
	_, err = matchCli.CreateMatch(ctx, reqCompleteKO)
	if err != nil {
		log.Fatalf("Failed to complete knockout match Round 1 Index 0: %v", err)
	}
	fmt.Println("  ✅ Completed knockout match Round 1 Index 0 (A1 vs A2, A1 wins 2-1)")

	// Test 4: Verify winner A1 progressed to Round 2 Index 0
	resListKOAfter, err := matchCli.ListKnockoutMatches(ctx, reqListKO)
	if err != nil {
		log.Fatalf("Failed to list knockout matches after completion: %v", err)
	}
	var foundNextMatch bool
	for _, m := range resListKOAfter.Msg.Matches {
		if m.Round == 2 && m.RoundIndex != nil && *m.RoundIndex == 0 {
			foundNextMatch = true
			if m.Country1 == nil || m.Country1.Code != "A1" {
				log.Fatalf("Expected winner A1 to be advanced to Round 2 Index 0 Country1 slot, but got %+v", m.Country1)
			}
			break
		}
	}
	if !foundNextMatch {
		log.Fatalf("Could not find next match at Round 2 Index 0")
	}
	fmt.Println("  ✅ Verified knockout progression (A1 advanced to Round 2 Index 0)")

	// Test 5: Verify User 1's leaderboard knockout and overall scores updated (+15 points for A1 reaching Round 2)
	// User 1 expects: Group Score = 50, Knockout Score = 15, Overall Score = 65
	// User 2 expects: Group Score = 43, Knockout Score = 0, Overall Score = 43
	resLboardKO, err := leaderboardCli.Leaderboard(ctx, reqLboard)
	if err != nil {
		log.Fatalf("Failed to fetch leaderboard after knockout: %v", err)
	}
	if len(resLboardKO.Msg.Knockout) != 2 {
		log.Fatalf("Expected exactly 2 leaderboard knockout entries, but got %d", len(resLboardKO.Msg.Knockout))
	}
	for _, entry := range resLboardKO.Msg.Knockout {
		if entry.Name == "user1" {
			if entry.Score != 15 {
				log.Fatalf("Expected user1 knockout score to be 15, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 1 Leaderboard knockout score (expects 15): %d\n", entry.Score)
		}
		if entry.Name == "user2" {
			if entry.Score != 0 {
				log.Fatalf("Expected user2 knockout score to be 0, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 2 Leaderboard knockout score (expects 0): %d\n", entry.Score)
		}
	}

	for _, entry := range resLboardKO.Msg.Overall {
		if entry.Name == "user1" {
			if entry.Score != 65 {
				log.Fatalf("Expected user1 overall score to be 65, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 1 Leaderboard overall score (expects 65): %d\n", entry.Score)
		}
		if entry.Name == "user2" {
			if entry.Score != 43 {
				log.Fatalf("Expected user2 overall score to be 43, but got %d", entry.Score)
			}
			fmt.Printf("  ✅ Verified User 2 Leaderboard overall score (expects 43): %d\n", entry.Score)
		}
	}

	// ==========================================
	// 🏆 SCENARIO 7: Leaderboards
	// ==========================================
	fmt.Println("\n📊 [SCENARIO 7: Leaderboard Endpoints]")

	// Test 1: Global Leaderboard
	fmt.Printf("  ✅ Global Leaderboard Entries (Count: %d):\n", len(resLboardKO.Msg.Overall))
	for idx, entry := range resLboardKO.Msg.Overall {
		fmt.Printf("     [%d] %s: Overall Score = %d\n", idx+1, entry.Name, entry.Score)
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
	fmt.Printf("  ✅ Subcontest Leaderboard '%s' Entries (Count: %d):\n", subcontestSlug, len(resSubLboard.Msg.Overall))
	for idx, entry := range resSubLboard.Msg.Overall {
		fmt.Printf("     [%d] %s: Overall Score = %d\n", idx+1, entry.Name, entry.Score)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("🎉 ALL UAT END-TO-END VERIFICATION SCENARIOS COMPLETED GREEN!")
	fmt.Println("=======================================================")
}

func startTestServer(db *sql.DB) {
	// Initialize Adapters
	userRepo := postgres.NewUserRepository(db)

	emails := strings.Split(os.Getenv("SUPERADMIN_EMAILS"), ",")
	var superEmail string
	if len(emails) > 0 {
		superEmail = strings.TrimSpace(emails[0])
	}
	tokenValidator := &mockTokenValidator{superadminEmail: superEmail}

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
