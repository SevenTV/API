package emotes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
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
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 86400, nil),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	return ctx.JSON(rest.SeeOther, []string{
		"/emotes/{emote}",
		"/emotes/global",
	})
}
