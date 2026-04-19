package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type PicksHandler struct {
	svc ports.PicksService
}

var _ v1connect.PicksServiceHandler = (*PicksHandler)(nil)

func NewPicksHandler(svc ports.PicksService) *PicksHandler {
	return &PicksHandler{svc: svc}
}

func (h *PicksHandler) ListGroupPicks(ctx context.Context, req *connect.Request[v1.ListGroupPicksRequest]) (*connect.Response[v1.ListGroupPicksResponse], error) {
	_, err := h.svc.ListGroupPicks(ctx, req.Msg.ContestSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListGroupPicksResponse{}), nil
}

func (h *PicksHandler) CreateGroupPicks(ctx context.Context, req *connect.Request[v1.CreateGroupPicksRequest]) (*connect.Response[v1.CreateGroupPicksResponse], error) {
	err := h.svc.CreateGroupPicks(ctx, req.Msg.ContestSlug, entity.GroupPick{}) // Mapping will happen here
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateGroupPicksResponse{}), nil
}

func (h *PicksHandler) ListKnockoutPicks(ctx context.Context, req *connect.Request[v1.ListKnockoutPicksRequest]) (*connect.Response[v1.ListKnockoutPicksResponse], error) {
	_, err := h.svc.ListKnockoutPicks(ctx, req.Msg.ContestSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListKnockoutPicksResponse{}), nil
}

func (h *PicksHandler) CreateKnockoutPicks(ctx context.Context, req *connect.Request[v1.CreateKnockoutPicksRequest]) (*connect.Response[v1.CreateKnockoutPicksResponse], error) {
	err := h.svc.CreateKnockoutPicks(ctx, req.Msg.ContestSlug, entity.KnockoutPick{}) // Mapping will happen here
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateKnockoutPicksResponse{}), nil
}
