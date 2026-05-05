package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/interceptor"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type UsersHandler struct {
	svc ports.UsersService
}

var _ v1connect.UsersServiceHandler = (*UsersHandler)(nil)

func NewUsersHandler(svc ports.UsersService) *UsersHandler {
	return &UsersHandler{svc: svc}
}

func (h *UsersHandler) CountUsers(ctx context.Context, req *connect.Request[v1.CountUsersRequest]) (*connect.Response[v1.CountUsersResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.CountUsersRequest]) (*connect.Response[v1.CountUsersResponse], error) {
		count, err := h.svc.CountUsers(ctx)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&v1.CountUsersResponse{
			Count: count,
		}), nil
	}
	return interceptor.WithPublic(handlerFunc)(ctx, req)
}

func (h *UsersHandler) DeleteUser(ctx context.Context, req *connect.Request[v1.DeleteUserRequest]) (*connect.Response[v1.DeleteUserResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.DeleteUserRequest]) (*connect.Response[v1.DeleteUserResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		err := h.svc.DeleteUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&v1.DeleteUserResponse{}), nil
	}
	return interceptor.WithAuth(handlerFunc)(ctx, req)
}
