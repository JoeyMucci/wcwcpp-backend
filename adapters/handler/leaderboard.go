package handler

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
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
	leaderboard, err := h.svc.Leaderboard(ctx, req.Msg.ContestSlug, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, err
	}

	groupLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["group"]))
	for _, l := range leaderboard["group"] {
		groupLB = append(groupLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}
	knockoutLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["knockout"]))
	for _, l := range leaderboard["knockout"] {
		knockoutLB = append(knockoutLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}
	overallLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["overall"]))
	for _, l := range leaderboard["overall"] {
		overallLB = append(overallLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}

	return connect.NewResponse(&v1.LeaderboardResponse{
		Group:    groupLB,
		Knockout: knockoutLB,
		Overall:  overallLB,
	}), nil
}

func (h *LeaderboardHandler) Subleaderboard(ctx context.Context, req *connect.Request[v1.SubleaderboardRequest]) (*connect.Response[v1.SubleaderboardResponse], error) {
	leaderboard, err := h.svc.Subleaderboard(ctx, req.Msg.SubcontestSlug, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, err
	}

	groupLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["group"]))
	for _, l := range leaderboard["group"] {
		groupLB = append(groupLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}
	knockoutLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["knockout"]))
	for _, l := range leaderboard["knockout"] {
		knockoutLB = append(knockoutLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}
	overallLB := make([]*v1.LeaderboardEntry, 0, len(leaderboard["overall"]))
	for _, l := range leaderboard["overall"] {
		overallLB = append(overallLB, &v1.LeaderboardEntry{
			Name:  l.Name,
			Score: l.Score,
		})
	}

	return connect.NewResponse(&v1.SubleaderboardResponse{
		Group:    groupLB,
		Knockout: knockoutLB,
		Overall:  overallLB,
	}), nil
}
