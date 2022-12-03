package emotes

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v2/model"
	"github.com/seventv/api/internal/global"
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
			newEmote(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

// @Summary Search Emotes
// @Description Search for emotes
// @Tags emotes
// @Produce json
// @Param query query string false "search by emote name / tags"
// @Success 200 {array} model.EmoteModel
// @Router /emotes [get]
func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	res := []model.Emote{{}}
	return ctx.JSON(rest.OK, &res)
}
