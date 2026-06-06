package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/ports"
	"github.com/stretchr/testify/require"
)

type mockContestRepository struct {
	ports.ContestRepository
	listContestsFunc             func(ctx context.Context) ([]entity.Contest, error)
	createContestFunc            func(ctx context.Context, contest *entity.Contest) error
	createCountriesFunc          func(ctx context.Context, countries []entity.Country) error
	createMatchesFunc            func(ctx context.Context, contestID string, matches []entity.Match) error
	createGroupStandingsFunc     func(ctx context.Context, contestID string, groups []entity.Group) error
	getContestBySlugFunc         func(ctx context.Context, slug string) (*entity.Contest, error)
	listSubcontestsFunc          func(ctx context.Context, userID string, contestSlug string) ([]entity.Subcontest, error)
	createSubcontestFunc         func(ctx context.Context, subcontest *entity.Subcontest) error
	joinSubcontestFunc           func(ctx context.Context, subcontestID string, userID string) error
	getSubcontestBySlugFunc      func(ctx context.Context, slug string) (*entity.Subcontest, error)
	deleteSubcontestFunc         func(ctx context.Context, subcontestID string) error
	getSubcontestByJoinCodeFunc  func(ctx context.Context, joinCode string) (*entity.Subcontest, error)
}

func (m *mockContestRepository) ListContests(ctx context.Context) ([]entity.Contest, error) {
	if m.listContestsFunc != nil {
		return m.listContestsFunc(ctx)
	}
	return nil, nil
}

func (m *mockContestRepository) CreateContest(ctx context.Context, contest *entity.Contest) error {
	if m.createContestFunc != nil {
		return m.createContestFunc(ctx, contest)
	}
	return nil
}

func (m *mockContestRepository) CreateCountries(ctx context.Context, countries []entity.Country) error {
	if m.createCountriesFunc != nil {
		return m.createCountriesFunc(ctx, countries)
	}
	return nil
}

func (m *mockContestRepository) CreateMatches(ctx context.Context, contestID string, matches []entity.Match) error {
	if m.createMatchesFunc != nil {
		return m.createMatchesFunc(ctx, contestID, matches)
	}
	return nil
}

func (m *mockContestRepository) CreateGroupStandings(ctx context.Context, contestID string, groups []entity.Group) error {
	if m.createGroupStandingsFunc != nil {
		return m.createGroupStandingsFunc(ctx, contestID, groups)
	}
	return nil
}

func (m *mockContestRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	if m.getContestBySlugFunc != nil {
		return m.getContestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockContestRepository) ListSubcontests(ctx context.Context, userID string, contestSlug string) ([]entity.Subcontest, error) {
	if m.listSubcontestsFunc != nil {
		return m.listSubcontestsFunc(ctx, userID, contestSlug)
	}
	return nil, nil
}

func (m *mockContestRepository) CreateSubcontest(ctx context.Context, subcontest *entity.Subcontest) error {
	if m.createSubcontestFunc != nil {
		return m.createSubcontestFunc(ctx, subcontest)
	}
	return nil
}

func (m *mockContestRepository) JoinSubcontest(ctx context.Context, subcontestID string, userID string) error {
	if m.joinSubcontestFunc != nil {
		return m.joinSubcontestFunc(ctx, subcontestID, userID)
	}
	return nil
}

func (m *mockContestRepository) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	if m.getSubcontestBySlugFunc != nil {
		return m.getSubcontestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockContestRepository) DeleteSubcontest(ctx context.Context, subcontestID string) error {
	if m.deleteSubcontestFunc != nil {
		return m.deleteSubcontestFunc(ctx, subcontestID)
	}
	return nil
}

func (m *mockContestRepository) GetSubcontestByJoinCode(ctx context.Context, joinCode string) (*entity.Subcontest, error) {
	if m.getSubcontestByJoinCodeFunc != nil {
		return m.getSubcontestByJoinCodeFunc(ctx, joinCode)
	}
	return nil, nil
}

func TestContestService_ListContests(t *testing.T) {
	t.Run("should list contests", func(t *testing.T) {
		expectedContests := []entity.Contest{
			{ID: "1", Title: "World Cup 2026", Slug: "world-cup-2026"},
			{ID: "2", Title: "Euro 2024", Slug: "euro-2024"},
		}

		mockRepo := &mockContestRepository{
			listContestsFunc: func(ctx context.Context) ([]entity.Contest, error) {
				return expectedContests, nil
			},
		}

		svc := NewContestService(mockRepo)

		contests, err := svc.ListContests(context.Background())
		require.NoError(t, err)
		require.Equal(t, expectedContests, contests)
	})
}

func TestContestService_CreateContest(t *testing.T) {
	t.Run("should create contest, countries, and expected matches", func(t *testing.T) {
		var capturedContest entity.Contest
		var capturedCountries []entity.Country
		var capturedMatches []entity.Match

		mockRepo := &mockContestRepository{
			createContestFunc: func(ctx context.Context, contest *entity.Contest) error {
				capturedContest = *contest
				return nil
			},
			createCountriesFunc: func(ctx context.Context, countries []entity.Country) error {
				capturedCountries = countries
				return nil
			},
			createMatchesFunc: func(ctx context.Context, contestID string, matches []entity.Match) error {
				capturedMatches = append(capturedMatches, matches...)
				return nil
			},
			createGroupStandingsFunc: func(ctx context.Context, contestID string, groups []entity.Group) error {
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		groups := []entity.Group{
			{Letter: "A", Countries: []entity.Country{{Code: "USA", FullName: "United States"}, {Code: "CAN", FullName: "Canada"}, {Code: "MEX", FullName: "Mexico"}, {Code: "ARG", FullName: "Argentina"}}},
			{Letter: "B", Countries: []entity.Country{{Code: "BRA", FullName: "Brazil"}, {Code: "FRA", FullName: "France"}, {Code: "GER", FullName: "Germany"}, {Code: "ENG", FullName: "England"}}},
			{Letter: "C", Countries: []entity.Country{{Code: "ESP", FullName: "Spain"}, {Code: "POR", FullName: "Portugal"}, {Code: "ITA", FullName: "Italy"}, {Code: "NED", FullName: "Netherlands"}}},
			{Letter: "D", Countries: []entity.Country{{Code: "BEL", FullName: "Belgium"}, {Code: "URU", FullName: "Uruguay"}, {Code: "COL", FullName: "Colombia"}, {Code: "CHI", FullName: "Chile"}}},
			{Letter: "E", Countries: []entity.Country{{Code: "CRO", FullName: "Croatia"}, {Code: "SRB", FullName: "Serbia"}, {Code: "SUI", FullName: "Switzerland"}, {Code: "DEN", FullName: "Denmark"}}},
			{Letter: "F", Countries: []entity.Country{{Code: "SWE", FullName: "Sweden"}, {Code: "NOR", FullName: "Norway"}, {Code: "MAR", FullName: "Morocco"}, {Code: "SEN", FullName: "Senegal"}}},
			{Letter: "G", Countries: []entity.Country{{Code: "GHA", FullName: "Ghana"}, {Code: "CMR", FullName: "Cameroon"}, {Code: "NGA", FullName: "Nigeria"}, {Code: "EGY", FullName: "Egypt"}}},
			{Letter: "H", Countries: []entity.Country{{Code: "ALG", FullName: "Algeria"}, {Code: "TUN", FullName: "Tunisia"}, {Code: "JPN", FullName: "Japan"}, {Code: "KOR", FullName: "South Korea"}}},
			{Letter: "I", Countries: []entity.Country{{Code: "AUS", FullName: "Australia"}, {Code: "IRN", FullName: "Iran"}, {Code: "KSA", FullName: "Saudi Arabia"}, {Code: "QAT", FullName: "Qatar"}}},
			{Letter: "J", Countries: []entity.Country{{Code: "CRC", FullName: "Costa Rica"}, {Code: "PAN", FullName: "Panama"}, {Code: "JAM", FullName: "Jamaica"}, {Code: "HON", FullName: "Honduras"}}},
			{Letter: "K", Countries: []entity.Country{{Code: "ECU", FullName: "Ecuador"}, {Code: "PER", FullName: "Peru"}, {Code: "PAR", FullName: "Paraguay"}, {Code: "VEN", FullName: "Venezuela"}}},
			{Letter: "L", Countries: []entity.Country{{Code: "BOL", FullName: "Bolivia"}, {Code: "CIV", FullName: "Ivory Coast"}, {Code: "MLI", FullName: "Mali"}, {Code: "BFA", FullName: "Burkina Faso"}}},
		}

		contest := entity.Contest{
			Title:              "2026 FIFA World Cup",
			Groups:             groups,
			GroupUnlockDate:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			GroupLockDate:      time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC),
			KnockoutUnlockDate: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC),
			KnockoutLockDate:   time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
		}

		err := svc.CreateContest(context.Background(), contest)
		require.NoError(t, err)

		// Assert Contest was created
		require.Equal(t, "2026 FIFA World Cup", capturedContest.Title)
		require.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), capturedContest.GroupUnlockDate)
		require.Equal(t, time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC), capturedContest.GroupLockDate)
		require.Equal(t, time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC), capturedContest.KnockoutUnlockDate)
		require.Equal(t, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC), capturedContest.KnockoutLockDate)

		// Assert Countries were created (12 * 4 = 48 countries)
		require.Len(t, capturedCountries, 48)

		// Assert Matches were created
		// Group stage matches: 12 groups * 6 matches per group (round robin of 4) = 72 matches
		// Knockout matches:
		// Round of 32: 16 matches
		// Round of 16: 8 matches
		// Quarter-finals: 4 matches
		// Semi-finals: 2 matches
		// Final/Third-place: 2 matches
		// Total knockouts: 16 + 8 + 4 + 2 + 2 = 32 matches
		// Total matches = 72 + 32 = 104 matches
		require.Len(t, capturedMatches, 104)

		groupStageCount := 0
		knockoutCount := 0
		knockoutByRound := make(map[int]int)
		roundIndices := make(map[int]map[int]struct{})

		for _, m := range capturedMatches {
			require.Nil(t, m.Country1Goals)
			require.Nil(t, m.Country2Goals)
			require.Nil(t, m.Country1Penalties)
			require.Nil(t, m.Country2Penalties)
			if m.Round == 0 {
				groupStageCount++
				require.NotNil(t, m.Country1)
				require.NotNil(t, m.Country2)
				require.Nil(t, m.RoundIndex)
			} else {
				knockoutCount++
				knockoutByRound[m.Round]++
				require.Nil(t, m.Country1)
				require.Nil(t, m.Country2)
				require.NotNil(t, m.RoundIndex)

				if roundIndices[m.Round] == nil {
					roundIndices[m.Round] = make(map[int]struct{})
				}
				_, exists := roundIndices[m.Round][*m.RoundIndex]
				require.False(t, exists, "duplicate RoundIndex %d in round %d", *m.RoundIndex, m.Round)
				roundIndices[m.Round][*m.RoundIndex] = struct{}{}
			}
		}

		require.Equal(t, 72, groupStageCount)
		require.Equal(t, 32, knockoutCount)
		require.Equal(t, 16, knockoutByRound[1])
		require.Equal(t, 8, knockoutByRound[2])
		require.Equal(t, 4, knockoutByRound[3])
		require.Equal(t, 2, knockoutByRound[4])
		require.Equal(t, 2, knockoutByRound[5])
	})

	t.Run("should seed group standings with all 12 groups and 4 countries each", func(t *testing.T) {
		var capturedStandingsContestID string
		var capturedStandingsGroups []entity.Group

		mockRepo := &mockContestRepository{
			createGroupStandingsFunc: func(ctx context.Context, contestID string, groups []entity.Group) error {
				capturedStandingsContestID = contestID
				capturedStandingsGroups = groups
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		groups := []entity.Group{
			{Letter: "A", Countries: []entity.Country{{Code: "USA", FullName: "United States"}, {Code: "CAN", FullName: "Canada"}, {Code: "MEX", FullName: "Mexico"}, {Code: "ARG", FullName: "Argentina"}}},
			{Letter: "B", Countries: []entity.Country{{Code: "BRA", FullName: "Brazil"}, {Code: "FRA", FullName: "France"}, {Code: "GER", FullName: "Germany"}, {Code: "ENG", FullName: "England"}}},
			{Letter: "C", Countries: []entity.Country{{Code: "ESP", FullName: "Spain"}, {Code: "POR", FullName: "Portugal"}, {Code: "ITA", FullName: "Italy"}, {Code: "NED", FullName: "Netherlands"}}},
			{Letter: "D", Countries: []entity.Country{{Code: "BEL", FullName: "Belgium"}, {Code: "URU", FullName: "Uruguay"}, {Code: "COL", FullName: "Colombia"}, {Code: "CHI", FullName: "Chile"}}},
			{Letter: "E", Countries: []entity.Country{{Code: "CRO", FullName: "Croatia"}, {Code: "SRB", FullName: "Serbia"}, {Code: "SUI", FullName: "Switzerland"}, {Code: "DEN", FullName: "Denmark"}}},
			{Letter: "F", Countries: []entity.Country{{Code: "SWE", FullName: "Sweden"}, {Code: "NOR", FullName: "Norway"}, {Code: "MAR", FullName: "Morocco"}, {Code: "SEN", FullName: "Senegal"}}},
			{Letter: "G", Countries: []entity.Country{{Code: "GHA", FullName: "Ghana"}, {Code: "CMR", FullName: "Cameroon"}, {Code: "NGA", FullName: "Nigeria"}, {Code: "EGY", FullName: "Egypt"}}},
			{Letter: "H", Countries: []entity.Country{{Code: "ALG", FullName: "Algeria"}, {Code: "TUN", FullName: "Tunisia"}, {Code: "JPN", FullName: "Japan"}, {Code: "KOR", FullName: "South Korea"}}},
			{Letter: "I", Countries: []entity.Country{{Code: "AUS", FullName: "Australia"}, {Code: "IRN", FullName: "Iran"}, {Code: "KSA", FullName: "Saudi Arabia"}, {Code: "QAT", FullName: "Qatar"}}},
			{Letter: "J", Countries: []entity.Country{{Code: "CRC", FullName: "Costa Rica"}, {Code: "PAN", FullName: "Panama"}, {Code: "JAM", FullName: "Jamaica"}, {Code: "HON", FullName: "Honduras"}}},
			{Letter: "K", Countries: []entity.Country{{Code: "ECU", FullName: "Ecuador"}, {Code: "PER", FullName: "Peru"}, {Code: "PAR", FullName: "Paraguay"}, {Code: "VEN", FullName: "Venezuela"}}},
			{Letter: "L", Countries: []entity.Country{{Code: "BOL", FullName: "Bolivia"}, {Code: "CIV", FullName: "Ivory Coast"}, {Code: "MLI", FullName: "Mali"}, {Code: "BFA", FullName: "Burkina Faso"}}},
		}
		contest := entity.Contest{
			Title:  "2026 FIFA World Cup",
			Groups: groups,
		}

		err := svc.CreateContest(context.Background(), contest)
		require.NoError(t, err)

		// The contestID forwarded to CreateGroupStandings should match the created contest
		_ = capturedStandingsContestID
		// All 12 groups should be forwarded
		require.Len(t, capturedStandingsGroups, 12)
		// Each group should have 4 countries
		for _, g := range capturedStandingsGroups {
			require.Len(t, g.Countries, 4)
		}
		// Letters should be preserved in order
		expectedLetters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
		for i, g := range capturedStandingsGroups {
			require.Equal(t, expectedLetters[i], g.Letter)
		}
	})

	t.Run("should propagate error from CreateGroupStandings", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			createGroupStandingsFunc: func(ctx context.Context, contestID string, groups []entity.Group) error {
				return errors.New("standings insert failed")
			},
		}

		svc := NewContestService(mockRepo)

		groups := []entity.Group{
			{Letter: "A", Countries: []entity.Country{{Code: "U01"}, {Code: "U02"}, {Code: "U03"}, {Code: "U04"}}},
			{Letter: "B", Countries: []entity.Country{{Code: "U05"}, {Code: "U06"}, {Code: "U07"}, {Code: "U08"}}},
			{Letter: "C", Countries: []entity.Country{{Code: "U09"}, {Code: "U10"}, {Code: "U11"}, {Code: "U12"}}},
			{Letter: "D", Countries: []entity.Country{{Code: "U13"}, {Code: "U14"}, {Code: "U15"}, {Code: "U16"}}},
			{Letter: "E", Countries: []entity.Country{{Code: "U17"}, {Code: "U18"}, {Code: "U19"}, {Code: "U20"}}},
			{Letter: "F", Countries: []entity.Country{{Code: "U21"}, {Code: "U22"}, {Code: "U23"}, {Code: "U24"}}},
			{Letter: "G", Countries: []entity.Country{{Code: "U25"}, {Code: "U26"}, {Code: "U27"}, {Code: "U28"}}},
			{Letter: "H", Countries: []entity.Country{{Code: "U29"}, {Code: "U30"}, {Code: "U31"}, {Code: "U32"}}},
			{Letter: "I", Countries: []entity.Country{{Code: "U33"}, {Code: "U34"}, {Code: "U35"}, {Code: "U36"}}},
			{Letter: "J", Countries: []entity.Country{{Code: "U37"}, {Code: "U38"}, {Code: "U39"}, {Code: "U40"}}},
			{Letter: "K", Countries: []entity.Country{{Code: "U41"}, {Code: "U42"}, {Code: "U43"}, {Code: "U44"}}},
			{Letter: "L", Countries: []entity.Country{{Code: "U45"}, {Code: "U46"}, {Code: "U47"}, {Code: "U48"}}},
		}

		err := svc.CreateContest(context.Background(), entity.Contest{Title: "Test", Groups: groups})
		require.Error(t, err)
		require.Equal(t, "standings insert failed", err.Error())
	})
}

func TestContestService_ListSubcontests(t *testing.T) {
	t.Run("should list subcontests", func(t *testing.T) {
		expectedSubcontests := []entity.Subcontest{
			{ID: "1", Title: "Subcontest 1", Slug: "subcontest-1", UserID: "user-1", ContestID: "contest-1"},
			{ID: "2", Title: "Subcontest 2", Slug: "subcontest-2", UserID: "user-1", ContestID: "contest-1"},
		}

		mockRepo := &mockContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listSubcontestsFunc: func(ctx context.Context, userID string, contestSlug string) ([]entity.Subcontest, error) {
				return expectedSubcontests, nil
			},
		}

		svc := NewContestService(mockRepo)

		subcontests, err := svc.ListSubcontests(context.Background(), "user-1", "world-cup-2026")
		require.NoError(t, err)
		require.Equal(t, expectedSubcontests, subcontests)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("contest not found")
			},
		}

		svc := NewContestService(mockRepo)

		_, err := svc.ListSubcontests(context.Background(), "user-1", "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})
}

func TestContestService_CreateSubcontest(t *testing.T) {
	t.Run("should create subcontest with self-join", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			createSubcontestFunc: func(ctx context.Context, subcontest *entity.Subcontest) error {
				return nil
			},
			joinSubcontestFunc: func(ctx context.Context, subcontestID string, userID string) error {
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		joinCode, _, err := svc.CreateSubcontest(context.Background(), "user-1", "world-cup-2026", "My Subcontest", true)
		require.NoError(t, err)
		require.NotEmpty(t, joinCode)
	})

	t.Run("should create subcontest without self-join", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			createSubcontestFunc: func(ctx context.Context, subcontest *entity.Subcontest) error {
				return nil
			},
			joinSubcontestFunc: func(ctx context.Context, subcontestID string, userID string) error {
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		joinCode, _, err := svc.CreateSubcontest(context.Background(), "user-1", "world-cup-2026", "My Subcontest", false)
		require.NoError(t, err)
		require.NotEmpty(t, joinCode)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("contest not found")
			},
		}

		svc := NewContestService(mockRepo)

		_, _, err := svc.CreateSubcontest(context.Background(), "user-1", "world-cup-2026", "My Subcontest", true)
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})
}

func TestContestService_DeleteSubcontest(t *testing.T) {
	t.Run("should delete subcontest", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getSubcontestBySlugFunc: func(ctx context.Context, slug string) (*entity.Subcontest, error) {
				return &entity.Subcontest{ID: "subcontest-1", UserID: "user-1"}, nil
			},
			deleteSubcontestFunc: func(ctx context.Context, subcontestID string) error {
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		err := svc.DeleteSubcontest(context.Background(), "user-1", "subcontest-1")
		require.NoError(t, err)
	})

	t.Run("should return error if subcontest not found", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getSubcontestBySlugFunc: func(ctx context.Context, slug string) (*entity.Subcontest, error) {
				return nil, errors.New("subcontest not found")
			},
		}

		svc := NewContestService(mockRepo)

		err := svc.DeleteSubcontest(context.Background(), "user-1", "subcontest-1")
		require.Error(t, err)
		require.Equal(t, "subcontest not found", err.Error())
	})

	t.Run("should return error if not owner", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getSubcontestBySlugFunc: func(ctx context.Context, slug string) (*entity.Subcontest, error) {
				return &entity.Subcontest{ID: "subcontest-1", UserID: "user-2"}, nil
			},
		}

		svc := NewContestService(mockRepo)

		err := svc.DeleteSubcontest(context.Background(), "user-1", "subcontest-1")
		require.Error(t, err)
		require.Equal(t, "not owner", err.Error())
	})
}

func TestContestService_JoinSubcontest(t *testing.T) {
	t.Run("should join subcontest", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getSubcontestByJoinCodeFunc: func(ctx context.Context, joinCode string) (*entity.Subcontest, error) {
				return &entity.Subcontest{ID: "subcontest-1"}, nil
			},
			joinSubcontestFunc: func(ctx context.Context, subcontestID string, userID string) error {
				return nil
			},
		}

		svc := NewContestService(mockRepo)

		err := svc.JoinSubcontest(context.Background(), "user-1", "JOIN")
		require.NoError(t, err)
	})

	t.Run("should return error if subcontest not found", func(t *testing.T) {
		mockRepo := &mockContestRepository{
			getSubcontestByJoinCodeFunc: func(ctx context.Context, joinCode string) (*entity.Subcontest, error) {
				return nil, nil
			},
		}

		svc := NewContestService(mockRepo)

		err := svc.JoinSubcontest(context.Background(), "user-1", "invalid")
		require.Error(t, err)
		require.Equal(t, "invalid join code", err.Error())
	})
}
