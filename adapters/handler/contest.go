package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/pkg/api/v1"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"github.com/joey/wcwcpp-backend/ports"
)

type ContestHandler struct {
	svc ports.ContestService
}

var _ v1connect.ContestServiceHandler = (*ContestHandler)(nil)

func NewContestHandler(svc ports.ContestService) *ContestHandler {
	return &ContestHandler{svc: svc}
}

func (h *ContestHandler) ListContests(ctx context.Context, req *connect.Request[v1.ListContestsRequest]) (*connect.Response[v1.ListContestsResponse], error) {
	_, err := h.svc.ListContests(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListContestsResponse{}), nil
}

func (h *ContestHandler) ListSubcontests(ctx context.Context, req *connect.Request[v1.ListSubcontestsRequest]) (*connect.Response[v1.ListSubcontestsResponse], error) {
	_, err := h.svc.ListSubcontests(ctx, req.Msg.ContestSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListSubcontestsResponse{}), nil
}

func (h *ContestHandler) CreateSubcontest(ctx context.Context, req *connect.Request[v1.CreateSubcontestRequest]) (*connect.Response[v1.CreateSubcontestResponse], error) {
	joinCode, err := h.svc.CreateSubcontest(ctx, req.Msg.ContestSlug, req.Msg.SubcontestTitle)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateSubcontestResponse{
		JoinCode: joinCode,
	}), nil
}

func (h *ContestHandler) DeleteSubcontest(ctx context.Context, req *connect.Request[v1.DeleteSubcontestRequest]) (*connect.Response[v1.DeleteSubcontestResponse], error) {
	err := h.svc.DeleteSubcontest(ctx, req.Msg.SubcontestSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DeleteSubcontestResponse{}), nil
}
