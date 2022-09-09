package routes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v3/routes/auth"
	"github.com/seventv/api/internal/rest/v3/routes/docs"
	emote_sets "github.com/seventv/api/internal/rest/v3/routes/emote-sets"
	"github.com/seventv/api/internal/rest/v3/routes/emotes"
	"github.com/seventv/api/internal/rest/v3/routes/users"
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
			emote_sets.New(r.Ctx),
			users.New(r.Ctx),
		},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 30, nil),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.Redirect("/v3/docs/ui", int(rest.Found))

	return nil
}

type Response struct {
	Online bool `json:"online"`
}
