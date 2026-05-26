package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/interceptor"
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
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.ListGroupMatchesRequest]) (*connect.Response[v1.ListGroupMatchesResponse], error) {
		matches, err := h.svc.ListGroupMatches(ctx, req.Msg.ContestSlug, req.Msg.Letter)
		if err != nil {
			return nil, err
		}

		return connect.NewResponse(&v1.ListGroupMatchesResponse{
			Matches: mapMatchesToProto(matches),
		}), nil
	}

	return interceptor.WithPublic(handlerFunc)(ctx, req)
}

func (h *MatchHandler) ListKnockoutMatches(ctx context.Context, req *connect.Request[v1.ListKnockoutMatchesRequest]) (*connect.Response[v1.ListKnockoutMatchesResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.ListKnockoutMatchesRequest]) (*connect.Response[v1.ListKnockoutMatchesResponse], error) {
		matches, err := h.svc.ListKnockoutMatches(ctx, req.Msg.ContestSlug)
		if err != nil {
			return nil, err
		}

		return connect.NewResponse(&v1.ListKnockoutMatchesResponse{
			Matches: mapMatchesToProto(matches),
		}), nil
	}

	return interceptor.WithPublic(handlerFunc)(ctx, req)
}

func (h *MatchHandler) CreateMatch(ctx context.Context, req *connect.Request[v1.CreateMatchRequest]) (*connect.Response[v1.CreateMatchResponse], error) {
	handlerFunc := func(ctx context.Context, req *connect.Request[v1.CreateMatchRequest]) (*connect.Response[v1.CreateMatchResponse], error) {
		err := h.svc.CreateMatch(ctx, req.Msg.ContestSlug, mapProtoToMatch(req.Msg.Match))
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&v1.CreateMatchResponse{}), nil
	}

	return interceptor.WithSuperadmin(handlerFunc)(ctx, req)
}

func mapMatchesToProto(matches []entity.Match) []*v1.Match {
	protoMatches := make([]*v1.Match, 0, len(matches))
	for _, m := range matches {
		var country1, country2 *v1.Country
		if m.Country1 != nil {
			country1 = &v1.Country{
				Code:     m.Country1.Code,
				FullName: m.Country1.FullName,
			}
		}
		if m.Country2 != nil {
			country2 = &v1.Country{
				Code:     m.Country2.Code,
				FullName: m.Country2.FullName,
			}
		}

		var c1g, c2g, c1p, c2p *int64
		if m.Country1Goals != nil {
			val := int64(*m.Country1Goals)
			c1g = &val
		}
		if m.Country2Goals != nil {
			val := int64(*m.Country2Goals)
			c2g = &val
		}
		if m.Country1Penalties != nil {
			val := int64(*m.Country1Penalties)
			c1p = &val
		}
		if m.Country2Penalties != nil {
			val := int64(*m.Country2Penalties)
			c2p = &val
		}

		var c1cs, c2cs *int64
		if m.Country1ConductScore != nil {
			val := int64(*m.Country1ConductScore)
			c1cs = &val
		}
		if m.Country2ConductScore != nil {
			val := int64(*m.Country2ConductScore)
			c2cs = &val
		}

		var roundIndex *int64
		if m.RoundIndex != nil {
			val := int64(*m.RoundIndex)
			roundIndex = &val
		}

		protoMatches = append(protoMatches, &v1.Match{
			Country1:             country1,
			Country2:             country2,
			Country1Goals:        c1g,
			Country2Goals:        c2g,
			Country1Penalties:    c1p,
			Country2Penalties:    c2p,
			Round:                int64(m.Round),
			RoundIndex:           roundIndex,
			Country1ConductScore: c1cs,
			Country2ConductScore: c2cs,
		})
	}
	return protoMatches
}

func mapProtoToMatch(pm *v1.Match) entity.Match {
	if pm == nil {
		return entity.Match{}
	}

	var country1, country2 *entity.Country
	if pm.Country1 != nil {
		country1 = &entity.Country{
			Code:     pm.Country1.Code,
			FullName: pm.Country1.FullName,
		}
	}
	if pm.Country2 != nil {
		country2 = &entity.Country{
			Code:     pm.Country2.Code,
			FullName: pm.Country2.FullName,
		}
	}

	var c1g, c2g, c1p, c2p *int
	if pm.Country1Goals != nil {
		val := int(*pm.Country1Goals)
		c1g = &val
	}
	if pm.Country2Goals != nil {
		val := int(*pm.Country2Goals)
		c2g = &val
	}
	if pm.Country1Penalties != nil {
		val := int(*pm.Country1Penalties)
		c1p = &val
	}
	if pm.Country2Penalties != nil {
		val := int(*pm.Country2Penalties)
		c2p = &val
	}

	var c1cs, c2cs *int
	if pm.Country1ConductScore != nil {
		val := int(*pm.Country1ConductScore)
		c1cs = &val
	}
	if pm.Country2ConductScore != nil {
		val := int(*pm.Country2ConductScore)
		c2cs = &val
	}

	var roundIndex *int
	if pm.RoundIndex != nil {
		val := int(*pm.RoundIndex)
		roundIndex = &val
	}

	return entity.Match{
		Country1:             country1,
		Country2:             country2,
		Country1Goals:        c1g,
		Country2Goals:        c2g,
		Country1Penalties:    c1p,
		Country2Penalties:    c2p,
		Round:                int(pm.Round),
		RoundIndex:           roundIndex,
		Country1ConductScore: c1cs,
		Country2ConductScore: c2cs,
	}
}
