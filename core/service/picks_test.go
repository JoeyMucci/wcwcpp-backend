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

type mockPicksRepository struct {
	ports.PicksRepository
	getContestBySlugFunc      func(ctx context.Context, slug string) (*entity.Contest, error)
	getSubcontestBySlugFunc   func(ctx context.Context, slug string) (*entity.Subcontest, error)
	listGroupPicksFunc        func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error)
	listGroupStandingsFunc    func(ctx context.Context, contestID string) ([]entity.GroupStanding, error)
	createGroupPicksFunc      func(ctx context.Context, userID string, contestID string, picks []entity.GroupPick) error
	listKnockoutPicksFunc     func(ctx context.Context, userID string, contestID string) (entity.KnockoutPick, error)
	listKnockoutResultsFunc   func(ctx context.Context, contestID string) (entity.KnockoutPick, error)
	createKnockoutPicksFunc   func(ctx context.Context, userID string, contestID string, pick entity.KnockoutPick) error
	getCountryCodeToIDMapFunc func(ctx context.Context) (map[string]string, error)
}

func (m *mockPicksRepository) GetContestBySlug(ctx context.Context, slug string) (*entity.Contest, error) {
	if m.getContestBySlugFunc != nil {
		return m.getContestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockPicksRepository) GetSubcontestBySlug(ctx context.Context, slug string) (*entity.Subcontest, error) {
	if m.getSubcontestBySlugFunc != nil {
		return m.getSubcontestBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockPicksRepository) ListGroupPicks(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
	if m.listGroupPicksFunc != nil {
		return m.listGroupPicksFunc(ctx, userID, contestID)
	}
	return nil, nil
}

func (m *mockPicksRepository) ListGroupStandings(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
	if m.listGroupStandingsFunc != nil {
		return m.listGroupStandingsFunc(ctx, contestID)
	}
	return nil, nil
}

func (m *mockPicksRepository) CreateGroupPicks(ctx context.Context, userID string, contestID string, picks []entity.GroupPick) error {
	if m.createGroupPicksFunc != nil {
		return m.createGroupPicksFunc(ctx, userID, contestID, picks)
	}
	return nil
}

func (m *mockPicksRepository) ListKnockoutPicks(ctx context.Context, userID string, contestID string) (entity.KnockoutPick, error) {
	if m.listKnockoutPicksFunc != nil {
		return m.listKnockoutPicksFunc(ctx, userID, contestID)
	}
	return entity.KnockoutPick{}, nil
}

func (m *mockPicksRepository) ListKnockoutResults(ctx context.Context, contestID string) (entity.KnockoutPick, error) {
	if m.listKnockoutResultsFunc != nil {
		return m.listKnockoutResultsFunc(ctx, contestID)
	}
	return entity.KnockoutPick{}, nil
}

func (m *mockPicksRepository) CreateKnockoutPicks(ctx context.Context, userID string, contestID string, pick entity.KnockoutPick) error {
	if m.createKnockoutPicksFunc != nil {
		return m.createKnockoutPicksFunc(ctx, userID, contestID, pick)
	}
	return nil
}

func (m *mockPicksRepository) GetCountryCodeToIDMap(ctx context.Context) (map[string]string, error) {
	if m.getCountryCodeToIDMapFunc != nil {
		return m.getCountryCodeToIDMapFunc(ctx)
	}
	return nil, nil
}

func TestPicksService_ListGroupPicks(t *testing.T) {
	t.Run("should return picks and standings on success", func(t *testing.T) {
		expectedPicks := []entity.GroupPick{
			{Letter: "A", Entries: []entity.GroupPickEntry{
				{Country: entity.Country{Code: "USA"}, Place: 1},
				{Country: entity.Country{Code: "MEX"}, Place: 2},
				{Country: entity.Country{Code: "CAN"}, Place: 3},
				{Country: entity.Country{Code: "ARG"}, Place: 4},
			}},
		}
		expectedStandings := []entity.GroupStanding{
			{Country: entity.Country{Code: "USA"}, Letter: "A", Points: 3},
		}

		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listGroupPicksFunc: func(ctx context.Context, userID string, contestID string) ([]entity.GroupPick, error) {
				require.Equal(t, "user-1", userID)
				require.Equal(t, "contest-1", contestID)
				return expectedPicks, nil
			},
			listGroupStandingsFunc: func(ctx context.Context, contestID string) ([]entity.GroupStanding, error) {
				require.Equal(t, "contest-1", contestID)
				return expectedStandings, nil
			},
		}

		svc := NewPicksService(repo)
		picks, standings, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.NoError(t, err)
		require.Equal(t, expectedPicks, picks)
		require.Equal(t, expectedStandings, standings)
	})

	t.Run("should return error if contest not found (nil)", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "nonexistent")
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})

	t.Run("should propagate error from contest lookup", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewPicksService(repo)
		_, _, err := svc.ListGroupPicks(context.Background(), "user-1", "world-cup-2026")
		require.Error(t, err)
		require.Equal(t, "db error", err.Error())
	})
}

func TestPicksService_CreateGroupPicks(t *testing.T) {
	t.Run("should save picks on success", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			createGroupPicksFunc: func(ctx context.Context, userID string, contestID string, picks []entity.GroupPick) error {
				require.Equal(t, "user-1", userID)
				require.Equal(t, "contest-1", contestID)
				require.Len(t, picks, 1)
				return nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateGroupPicks(context.Background(), "user-1", "world-cup-2026", []entity.GroupPick{{Letter: "A"}})
		require.NoError(t, err)
	})

	t.Run("should return error if contest not found", func(t *testing.T) {
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return nil, nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateGroupPicks(context.Background(), "user-1", "world-cup-2026", []entity.GroupPick{{Letter: "A"}})
		require.Error(t, err)
		require.Equal(t, "contest not found", err.Error())
	})

	t.Run("should return error if group stage picks are locked (before unlock)", func(t *testing.T) {
		importTime := time.Now()
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{
					ID:              "contest-1",
					GroupUnlockDate: importTime.Add(24 * time.Hour),
					GroupLockDate:   importTime.Add(48 * time.Hour),
				}, nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateGroupPicks(context.Background(), "user-1", "world-cup-2026", []entity.GroupPick{{Letter: "A"}})
		require.Error(t, err)
		require.Equal(t, "group stage picks are locked", err.Error())
	})

	t.Run("should return error if group stage picks are locked (after lock)", func(t *testing.T) {
		importTime := time.Now()
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{
					ID:              "contest-1",
					GroupUnlockDate: importTime.Add(-48 * time.Hour),
					GroupLockDate:   importTime.Add(-24 * time.Hour),
				}, nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateGroupPicks(context.Background(), "user-1", "world-cup-2026", []entity.GroupPick{{Letter: "A"}})
		require.Error(t, err)
		require.Equal(t, "group stage picks are locked", err.Error())
	})
}

func TestPicksService_ListKnockoutPicks(t *testing.T) {
	t.Run("should return picks and results on success", func(t *testing.T) {
		samplePick := entity.KnockoutPick{Entries: []entity.KnockoutPickEntry{{Round: 16}}}
		sampleResult := entity.KnockoutPick{Entries: []entity.KnockoutPickEntry{{Round: 8}}}

		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			listKnockoutPicksFunc: func(ctx context.Context, userID string, contestID string) (entity.KnockoutPick, error) {
				require.Equal(t, "user-1", userID)
				require.Equal(t, "contest-1", contestID)
				return samplePick, nil
			},
			listKnockoutResultsFunc: func(ctx context.Context, contestID string) (entity.KnockoutPick, error) {
				require.Equal(t, "contest-1", contestID)
				return sampleResult, nil
			},
		}

		svc := NewPicksService(repo)
		pick, result, err := svc.ListKnockoutPicks(context.Background(), "user-1", "world-cup-2026")
		require.NoError(t, err)
		require.Equal(t, samplePick, pick)
		require.Equal(t, sampleResult, result)
	})
}

func TestPicksService_CreateKnockoutPicks(t *testing.T) {
	t.Run("should save picks on success", func(t *testing.T) {
		samplePick := entity.KnockoutPick{Entries: []entity.KnockoutPickEntry{{Round: 16}}}

		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{ID: "contest-1"}, nil
			},
			createKnockoutPicksFunc: func(ctx context.Context, userID string, contestID string, pick entity.KnockoutPick) error {
				require.Equal(t, "user-1", userID)
				require.Equal(t, "contest-1", contestID)
				require.Equal(t, samplePick, pick)
				return nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateKnockoutPicks(context.Background(), "user-1", "world-cup-2026", samplePick)
		require.NoError(t, err)
	})

	t.Run("should return error if knockout stage picks are locked (before unlock)", func(t *testing.T) {
		importTime := time.Now()
		samplePick := entity.KnockoutPick{Entries: []entity.KnockoutPickEntry{{Round: 16}}}
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{
					ID:                 "contest-1",
					KnockoutUnlockDate: importTime.Add(24 * time.Hour),
					KnockoutLockDate:   importTime.Add(48 * time.Hour),
				}, nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateKnockoutPicks(context.Background(), "user-1", "world-cup-2026", samplePick)
		require.Error(t, err)
		require.Equal(t, "knockout stage picks are locked", err.Error())
	})

	t.Run("should return error if knockout stage picks are locked (after lock)", func(t *testing.T) {
		importTime := time.Now()
		samplePick := entity.KnockoutPick{Entries: []entity.KnockoutPickEntry{{Round: 16}}}
		repo := &mockPicksRepository{
			getContestBySlugFunc: func(ctx context.Context, slug string) (*entity.Contest, error) {
				return &entity.Contest{
					ID:                 "contest-1",
					KnockoutUnlockDate: importTime.Add(-48 * time.Hour),
					KnockoutLockDate:   importTime.Add(-24 * time.Hour),
				}, nil
			},
		}

		svc := NewPicksService(repo)
		err := svc.CreateKnockoutPicks(context.Background(), "user-1", "world-cup-2026", samplePick)
		require.Error(t, err)
		require.Equal(t, "knockout stage picks are locked", err.Error())
	})
}
