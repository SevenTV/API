package emote_sets

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
)

type Route struct {
	Ctx global.Context
}

func New(gctx global.Context) Route {
	return Route{gctx}
}

func (r Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/emote-sets",
		Method: rest.GET,
		Children: []rest.Route{
			newEmoteSetByIDRoute(r.Ctx),
		},
	}
}

// @Summary Search Emote Sets
// @Description Search for Emote Sets
// @Tags emote-sets
// @Produce json
// @Param query query string false "search by emote set name / tags"
// @Success 200 {array} model.EmoteSetModel
// @Router /emote-sets [get]
func (r Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrUnknownRoute().SetDetail("This route is not implemented yet")
}
