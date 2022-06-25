package routes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/middleware"
	"github.com/seventv/api/internal/rest/v3/routes/auth"
	"github.com/seventv/api/internal/rest/v3/routes/docs"
	"github.com/seventv/api/internal/rest/v3/routes/emotes"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/v3" + r.Ctx.Config().Http.VersionSuffix,
		Method: rest.GET,
		Children: []rest.Route{
			docs.New(r.Ctx),
			auth.New(r.Ctx),
			emotes.New(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 30, nil),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return ctx.JSON(rest.OK, &Response{
		Online: true,
	})
}

type Response struct {
	Online bool `json:"online"`
}
