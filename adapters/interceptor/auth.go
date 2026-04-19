package interceptor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
)

type userContextKey struct{}
type emailContextKey struct{}

// WithAuth is a wrapper for a ConnectRPC handler that requires authentication.
func WithAuth[Req any, Res any](next func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error)) func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error) {
	return func(ctx context.Context, req *connect.Request[Req]) (*connect.Response[Res], error) {
		userID, email, err := validateAuthHeader(req.Header().Get("Authorization"))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = context.WithValue(ctx, userContextKey{}, userID)
		ctx = context.WithValue(ctx, emailContextKey{}, email)

		return next(ctx, req)
	}
}

// WithSuperadmin is a wrapper for a ConnectRPC handler that requires superadmin privileges.
func WithSuperadmin[Req any, Res any](next func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error)) func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error) {
	return func(ctx context.Context, req *connect.Request[Req]) (*connect.Response[Res], error) {
		userID, email, err := validateAuthHeader(req.Header().Get("Authorization"))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		if !isSuperadmin(email) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("superadmin access required"))
		}

		ctx = context.WithValue(ctx, userContextKey{}, userID)
		ctx = context.WithValue(ctx, emailContextKey{}, email)

		return next(ctx, req)
	}
}

// WithPublic is a wrapper for public endpoints. It doesn't enforce authentication,
// but it is provided for consistency when defining routes.
func WithPublic[Req any, Res any](next func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error)) func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error) {
	return next
}

func validateAuthHeader(authHeader string) (string, string, error) {
	if authHeader == "" {
		return "", "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", "", errors.New("invalid authorization header format")
	}

	tokenString := parts[1]
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback_secret_for_dev_only"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}

	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	if userID == "" || email == "" {
		return "", "", errors.New("missing user details in token")
	}

	return userID, email, nil
}

func isSuperadmin(email string) bool {
	superadminsEnv := os.Getenv("SUPERADMIN_EMAILS")
	if superadminsEnv == "" {
		return false
	}

	superadmins := strings.Split(superadminsEnv, ",")
	for _, admin := range superadmins {
		if strings.TrimSpace(admin) == email {
			return true
		}
	}
	return false
}

// GetUserID retrieves the user ID from the context if it exists.
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userContextKey{}).(string)
	return id, ok
}

// GetEmail retrieves the email from the context if it exists.
func GetEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(emailContextKey{}).(string)
	return email, ok
}
