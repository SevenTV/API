package docs

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v3/docs"
	"github.com/seventv/api/internal/global"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/docs",
		Method: rest.GET,
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.SetBodyString(docs.SwaggerInfo.ReadDoc())
	ctx.SetContentType("application/json")

	return nil
}
