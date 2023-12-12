package entitlements

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
)

type entitlementRoute struct {
	gctx global.Context
}

func New(gctx global.Context) rest.Route {
	return &entitlementRoute{gctx}
}

func (r *entitlementRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/entitlements",
		Method: rest.GET,
		Children: []rest.Route{
			newCreate(r.gctx),
		},
		Middleware: []rest.Middleware{},
	}
}

func (r *entitlementRoute) Handler(ctx *rest.Ctx) rest.APIError {
	return nil
}
