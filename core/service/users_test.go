package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsersService_CountUsers(t *testing.T) {
	tests := []struct {
		name        string
		mockCount   int64
		mockErr     error
		expectCount int64
		expectErr   bool
	}{
		{
			name:        "success",
			mockCount:   10,
			mockErr:     nil,
			expectCount: 10,
			expectErr:   false,
		},
		{
			name:        "error",
			mockCount:   0,
			mockErr:     errors.New("db error"),
			expectCount: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{
				countUsersFunc: func(ctx context.Context) (int64, error) {
					return tt.mockCount, tt.mockErr
				},
			}
			svc := NewUsersService(repo)

			count, err := svc.CountUsers(context.Background())
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCount, count)
			}
		})
	}
}
