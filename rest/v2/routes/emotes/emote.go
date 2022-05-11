package emotes

import (
	"github.com/SevenTV/Common/errors"
	"github.com/seventv/api/global"
	"github.com/seventv/api/rest/rest"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/emotes",
		Method: rest.GET,
		Children: []rest.Route{
			newEmote(r.Ctx),
			newGlobals(r.Ctx),
		},
		Middleware: []func(ctx *rest.Ctx) errors.APIError{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	return ctx.JSON(rest.SeeOther, []string{
		"/emotes/{emote}",
		"/emotes/global",
	})
}
