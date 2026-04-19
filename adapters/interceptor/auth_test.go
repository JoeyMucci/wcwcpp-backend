package interceptor

import (
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func generateTestToken(secret, userID, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestWithAuth(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")

	validToken := generateTestToken("test_secret", "user123", "test@example.com")
	invalidSigToken := generateTestToken("wrong_secret", "user123", "test@example.com")

	dummyHandler := func(ctx context.Context, req *connect.Request[string]) (*connect.Response[string], error) {
		userID, ok := GetUserID(ctx)
		require.True(t, ok)
		require.Equal(t, "user123", userID)

		email, ok := GetEmail(ctx)
		require.True(t, ok)
		require.Equal(t, "test@example.com", email)

		okMsg := "ok"
		return connect.NewResponse(&okMsg), nil
	}

	wrapped := WithAuth(dummyHandler)

	t.Run("Valid Token", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)
		req.Header().Set("Authorization", "Bearer "+validToken)

		res, err := wrapped(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, "ok", *res.Msg)
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)
		req.Header().Set("Authorization", "Bearer "+invalidSigToken)

		_, err := wrapped(context.Background(), req)
		require.Error(t, err)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})

	t.Run("Missing Header", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)

		_, err := wrapped(context.Background(), req)
		require.Error(t, err)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	})
}

func TestWithSuperadmin(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	os.Setenv("SUPERADMIN_EMAILS", "super1@example.com, super2@example.com")

	superToken := generateTestToken("test_secret", "user1", "super1@example.com")
	normalToken := generateTestToken("test_secret", "user2", "normal@example.com")

	dummyHandler := func(ctx context.Context, req *connect.Request[string]) (*connect.Response[string], error) {
		okMsg := "ok"
		return connect.NewResponse(&okMsg), nil
	}

	wrapped := WithSuperadmin(dummyHandler)

	t.Run("Valid Superadmin", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)
		req.Header().Set("Authorization", "Bearer "+superToken)

		res, err := wrapped(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, "ok", *res.Msg)
	})

	t.Run("Not Superadmin", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)
		req.Header().Set("Authorization", "Bearer "+normalToken)

		_, err := wrapped(context.Background(), req)
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	})
}

func TestWithPublic(t *testing.T) {
	dummyHandler := func(ctx context.Context, req *connect.Request[string]) (*connect.Response[string], error) {
		okMsg := "ok"
		return connect.NewResponse(&okMsg), nil
	}

	wrapped := WithPublic(dummyHandler)

	t.Run("Passes Through", func(t *testing.T) {
		pingMsg := "ping"
		req := connect.NewRequest(&pingMsg)
		// Intentionally no headers set to ensure it's completely public

		res, err := wrapped(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, "ok", *res.Msg)
	})
}
