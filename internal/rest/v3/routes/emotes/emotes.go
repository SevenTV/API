package emotes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/model"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	listen(gCtx)
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/emotes",
		Method: rest.GET,
		Children: []rest.Route{
			newCreate(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

// Emote Search
// @Summary Search Emotes
// @Description Search for emotes
// @Tags emotes
// @Produce json
// @Param query query string false "search by emote name / tags"
// @Success 200 {array} model.Emote
// @Router /emotes [get]
func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	res := []model.Emote{{}}
	return ctx.JSON(rest.OK, &res)
}
