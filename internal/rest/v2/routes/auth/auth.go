package auth

import (
	"fmt"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/auth",
		Method: rest.GET,
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.Redirect(fmt.Sprintf("/v3%s/auth/twitch?old=true", r.Ctx.Config().Http.VersionSuffix), int(rest.Found))
	return nil
}
