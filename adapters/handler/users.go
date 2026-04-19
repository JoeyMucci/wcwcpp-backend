package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
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
	count, err := h.svc.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CountUsersResponse{
		Count: count,
	}), nil
}
