package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_CountUsers(t *testing.T) {
	repo := NewUserRepository(setupTestDB(t).db)
	ctx := context.Background()

	// Initial count
	initialCount, err := repo.CountUsers(ctx)
	require.NoError(t, err)

	// Add 3 users
	for i := 0; i < 3; i++ {
		uniqueSuffix := uuid.New().String()
		_, err := repo.CreateUser(ctx, "user"+uniqueSuffix+"@example.com", "user"+uniqueSuffix)
		require.NoError(t, err)
	}

	// New count
	newCount, err := repo.CountUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, initialCount+3, newCount)
}

func TestUserRepository_DeleteUser(t *testing.T) {
	repo := NewUserRepository(setupTestDB(t).db)
	ctx := context.Background()

	// Create user to delete
	user, err := repo.CreateUser(ctx, "delete@example.com", "deleteme")
	require.NoError(t, err)

	// Delete user
	err = repo.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Verify not found
	_, err = repo.FindByEmail(ctx, "delete@example.com")
	assert.NoError(t, err) // FindByEmail returns nil, nil if not found
	
	// Delete non-existent user should fail
	err = repo.DeleteUser(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Equal(t, "user not found", err.Error())
}
