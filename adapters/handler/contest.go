package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/interceptor"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
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
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.ListContestsRequest]) (*connect.Response[v1.ListContestsResponse], error) {
		contests, err := h.svc.ListContests(ctx)
		if err != nil {
			return nil, err
		}

		var pbContests []*v1.Contest
		now := time.Now()
		for _, c := range contests {
			isActive := (now.After(c.GroupUnlockDate) && now.Before(c.GroupLockDate)) ||
				(now.After(c.KnockoutUnlockDate) && now.Before(c.KnockoutLockDate))

			pbContests = append(pbContests, &v1.Contest{
				Title:  c.Title,
				Slug:   c.Slug,
				Active: isActive,
			})
		}

		return connect.NewResponse(&v1.ListContestsResponse{
			Contests: pbContests,
		}), nil
	}

	return interceptor.WithPublic(handlerFunc)(ctx, req)
}

func (h *ContestHandler) CreateContest(ctx context.Context, req *connect.Request[v1.CreateContestRequest]) (*connect.Response[v1.CreateContestResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.CreateContestRequest]) (*connect.Response[v1.CreateContestResponse], error) {
		var groups []entity.Group
		for _, g := range req.Msg.Groups {
			var countries []entity.Country
			for _, c := range g.Countries {
				countries = append(countries, entity.Country{
					Code:     c.Code,
					FullName: c.FullName,
				})
			}
			groups = append(groups, entity.Group{
				Letter:    g.Letter,
				Countries: countries,
			})
		}

		contest := entity.Contest{
			Title:  req.Msg.Title,
			Groups: groups,
		}

		if req.Msg.GroupUnlockDate != nil {
			contest.GroupUnlockDate = req.Msg.GroupUnlockDate.AsTime()
		}
		if req.Msg.GroupLockDate != nil {
			contest.GroupLockDate = req.Msg.GroupLockDate.AsTime()
		}
		if req.Msg.KnockoutUnlockDate != nil {
			contest.KnockoutUnlockDate = req.Msg.KnockoutUnlockDate.AsTime()
		}
		if req.Msg.KnockoutLockDate != nil {
			contest.KnockoutLockDate = req.Msg.KnockoutLockDate.AsTime()
		}

		err := h.svc.CreateContest(ctx, contest)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("a contest with this title already exists"))
			}
			return nil, err
		}
		return connect.NewResponse(&v1.CreateContestResponse{}), nil
	}

	return interceptor.WithSuperadmin(handlerFunc)(ctx, req)
}

func (h *ContestHandler) ListSubcontests(ctx context.Context, req *connect.Request[v1.ListSubcontestsRequest]) (*connect.Response[v1.ListSubcontestsResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.ListSubcontestsRequest]) (*connect.Response[v1.ListSubcontestsResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		subcontests, err := h.svc.ListSubcontests(ctx, userID, req.Msg.ContestSlug)
		if err != nil {
			return nil, err
		}

		var pbSubcontests []*v1.Subcontest
		for _, s := range subcontests {
			pbSubcontests = append(pbSubcontests, &v1.Subcontest{
				Title:    s.Title,
				Slug:     s.Slug,
				IsOwner:  s.IsOwner,
				IsMember: s.IsMember,
			})
		}

		return connect.NewResponse(&v1.ListSubcontestsResponse{
			Subcontests: pbSubcontests,
		}), nil
	}
	return interceptor.WithAuth(handlerFunc)(ctx, req)
}

func (h *ContestHandler) CreateSubcontest(ctx context.Context, req *connect.Request[v1.CreateSubcontestRequest]) (*connect.Response[v1.CreateSubcontestResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.CreateSubcontestRequest]) (*connect.Response[v1.CreateSubcontestResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		joinCode, err := h.svc.CreateSubcontest(ctx, userID, req.Msg.ContestSlug, req.Msg.SubcontestTitle, req.Msg.SelfJoin)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("a subcontest with this title or join code already exists"))
			}
			return nil, err
		}
		return connect.NewResponse(&v1.CreateSubcontestResponse{
			JoinCode: joinCode,
		}), nil
	}
	return interceptor.WithAuth(handlerFunc)(ctx, req)
}

func (h *ContestHandler) DeleteSubcontest(ctx context.Context, req *connect.Request[v1.DeleteSubcontestRequest]) (*connect.Response[v1.DeleteSubcontestResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.DeleteSubcontestRequest]) (*connect.Response[v1.DeleteSubcontestResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		err := h.svc.DeleteSubcontest(ctx, userID, req.Msg.SubcontestSlug)
		if err != nil {
			if err.Error() == "subcontest not found" {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			if err.Error() == "not owner" {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			return nil, err
		}
		return connect.NewResponse(&v1.DeleteSubcontestResponse{}), nil
	}
	return interceptor.WithAuth(handlerFunc)(ctx, req)
}

func (h *ContestHandler) JoinSubcontest(ctx context.Context, req *connect.Request[v1.JoinSubcontestRequest]) (*connect.Response[v1.JoinSubcontestResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.JoinSubcontestRequest]) (*connect.Response[v1.JoinSubcontestResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		err := h.svc.JoinSubcontest(ctx, userID, req.Msg.JoinCode)
		if err != nil {
			if err.Error() == "invalid join code" {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("invalid join code"))
			}
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("already joined this subcontest"))
			}
			return nil, err
		}
		return connect.NewResponse(&v1.JoinSubcontestResponse{}), nil
	}
	return interceptor.WithAuth(handlerFunc)(ctx, req)
}

func (h *ContestHandler) FinalizeGroupRankings(ctx context.Context, req *connect.Request[v1.FinalizeGroupRankingsRequest]) (*connect.Response[v1.FinalizeGroupRankingsResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.FinalizeGroupRankingsRequest]) (*connect.Response[v1.FinalizeGroupRankingsResponse], error) {
		err := h.svc.FinalizeGroupRankings(ctx, req.Msg.ContestSlug, req.Msg.GroupLetter, req.Msg.OrderedCountryCodes)
		if err != nil {
			if strings.Contains(err.Error(), "incomplete group matches") || strings.Contains(err.Error(), "already been finalized") {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			if err.Error() == "contest not found" {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, err
		}
		return connect.NewResponse(&v1.FinalizeGroupRankingsResponse{}), nil
	}
	return interceptor.WithSuperadmin(handlerFunc)(ctx, req)
}

func (h *ContestHandler) FinalizeThirdPlaceQualifier(ctx context.Context, req *connect.Request[v1.FinalizeThirdPlaceQualifierRequest]) (*connect.Response[v1.FinalizeThirdPlaceQualifierResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.FinalizeThirdPlaceQualifierRequest]) (*connect.Response[v1.FinalizeThirdPlaceQualifierResponse], error) {
		err := h.svc.FinalizeThirdPlaceQualifier(ctx, req.Msg.ContestSlug, req.Msg.GroupLetter, req.Msg.IsWildcardQualifier)
		if err != nil {
			if strings.Contains(err.Error(), "already been finalized") || strings.Contains(err.Error(), "rankings might not be finalized") {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			if err.Error() == "contest not found" {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, err
		}
		return connect.NewResponse(&v1.FinalizeThirdPlaceQualifierResponse{}), nil
	}
	return interceptor.WithSuperadmin(handlerFunc)(ctx, req)
}
