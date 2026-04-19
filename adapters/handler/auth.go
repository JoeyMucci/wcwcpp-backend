package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/service"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type AuthHandler struct {
	svc ports.AuthService
}

var _ v1connect.AuthServiceHandler = (*AuthHandler)(nil)

func NewAuthHandler(svc ports.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	token, user, err := h.svc.Login(ctx, req.Msg.GoogleIdToken, req.Msg.Username)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, service.ErrInvalidToken) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.LoginResponse{
		AccessToken: token,
		User: &v1.User{
			Id:       user.ID,
			Email:    user.Email,
			Username: user.Username,
		},
	}), nil
}
