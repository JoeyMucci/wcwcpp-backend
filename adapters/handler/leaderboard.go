package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type LeaderboardHandler struct {
	svc ports.LeaderboardService
}

var _ v1connect.LeaderboardServiceHandler = (*LeaderboardHandler)(nil)

func NewLeaderboardHandler(svc ports.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{svc: svc}
}

func (h *LeaderboardHandler) Leaderboard(ctx context.Context, req *connect.Request[v1.LeaderboardRequest]) (*connect.Response[v1.LeaderboardResponse], error) {
	_, nextPageToken, err := h.svc.Leaderboard(ctx, req.Msg.ContestSlug, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.LeaderboardResponse{
		NextPageToken: nextPageToken,
	}), nil
}

func (h *LeaderboardHandler) Subleaderboard(ctx context.Context, req *connect.Request[v1.SubleaderboardRequest]) (*connect.Response[v1.SubleaderboardResponse], error) {
	_, nextPageToken, err := h.svc.Subleaderboard(ctx, req.Msg.SubcontestSlug, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.SubleaderboardResponse{
		NextPageToken: nextPageToken,
	}), nil
}
