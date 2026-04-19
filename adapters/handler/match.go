package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/core/entity"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type MatchHandler struct {
	svc ports.MatchService
}

var _ v1connect.MatchServiceHandler = (*MatchHandler)(nil)

func NewMatchHandler(svc ports.MatchService) *MatchHandler {
	return &MatchHandler{svc: svc}
}

func (h *MatchHandler) ListGroupMatches(ctx context.Context, req *connect.Request[v1.ListGroupMatchesRequest]) (*connect.Response[v1.ListGroupMatchesResponse], error) {
	_, err := h.svc.ListGroupMatches(ctx, req.Msg.ContestSlug, req.Msg.Letter)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListGroupMatchesResponse{}), nil
}

func (h *MatchHandler) ListKnockoutMatches(ctx context.Context, req *connect.Request[v1.ListKnockoutMatchesRequest]) (*connect.Response[v1.ListKnockoutMatchesResponse], error) {
	_, err := h.svc.ListKnockoutMatches(ctx, req.Msg.ContestSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListKnockoutMatchesResponse{}), nil
}

func (h *MatchHandler) CreateMatch(ctx context.Context, req *connect.Request[v1.CreateMatchRequest]) (*connect.Response[v1.CreateMatchResponse], error) {
	err := h.svc.CreateMatch(ctx, req.Msg.ContestSlug, entity.Match{}) // Mapping will happen here
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateMatchResponse{}), nil
}
