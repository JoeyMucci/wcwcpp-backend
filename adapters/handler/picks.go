package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/interceptor"
	"github.com/joey/wcwcpp-backend/core/entity"
	v1 "github.com/joey/wcwcpp-backend/pkg/api/v1"
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
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.ListGroupPicksRequest]) (*connect.Response[v1.ListGroupPicksResponse], error) {
		userID, ok := interceptor.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		picks, standings, err := h.svc.ListGroupPicks(ctx, userID, req.Msg.ContestSlug)
		if err != nil {
			if err.Error() == "contest not found" {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, err
		}

		pbPicks := make([]*v1.GroupPick, 0, len(picks))
		for _, p := range picks {
			pbCountries := make([]*v1.Country, 0, len(p.Entries))
			for _, e := range p.Entries {
				pbCountries = append(pbCountries, &v1.Country{
					Code:     e.Country.Code,
					FullName: e.Country.FullName,
				})
			}
			pbPicks = append(pbPicks, &v1.GroupPick{
				Group: &v1.Group{
					Letter:    p.Letter,
					Countries: pbCountries,
				},
				ExtraQualifier: p.ExtraQualifier,
			})
		}

		pbRankedGroups := buildRankedGroups(standings)

		return connect.NewResponse(&v1.ListGroupPicksResponse{
			Picks:        pbPicks,
			RankedGroups: pbRankedGroups,
		}), nil
	}

	return interceptor.WithAuth(handlerFunc)(ctx, req)
}

// buildRankedGroups converts a flat slice of GroupStanding (sorted by letter, then points desc)
// into a slice of RankedGroup, one per letter.
func buildRankedGroups(standings []entity.GroupStanding) []*v1.RankedGroup {
	groupMap := make(map[string]*v1.RankedGroup)
	order := make([]string, 0)

	for _, s := range standings {
		if _, exists := groupMap[s.Letter]; !exists {
			groupMap[s.Letter] = &v1.RankedGroup{Letter: s.Letter}
			order = append(order, s.Letter)
		}
		groupMap[s.Letter].RankedCountries = append(groupMap[s.Letter].RankedCountries, &v1.RankedCountry{
			Code:           s.Country.Code,
			FullName:       s.Country.FullName,
			Points:         s.Points,
			Wins:           s.Wins,
			Draws:          s.Draws,
			Losses:         s.Losses,
			GoalsFor:       s.GoalsFor,
			GoalsAgainst:   s.GoalsAgainst,
			GoalDifference: s.GoalDifference,
			ConductScore:   s.ConductScore,
		})
	}

	result := make([]*v1.RankedGroup, 0, len(order))
	for _, letter := range order {
		result = append(result, groupMap[letter])
	}
	return result
}

func (h *PicksHandler) CreateGroupPicks(ctx context.Context, req *connect.Request[v1.CreateGroupPicksRequest]) (*connect.Response[v1.CreateGroupPicksResponse], error) {
	err := h.svc.CreateGroupPicks(ctx, req.Msg.ContestSlug, entity.GroupPick{})
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
	err := h.svc.CreateKnockoutPicks(ctx, req.Msg.ContestSlug, entity.KnockoutPick{})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateKnockoutPicksResponse{}), nil
}
