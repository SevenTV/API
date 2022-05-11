package routes

import (
	"github.com/SevenTV/Common/errors"
	"github.com/seventv/api/global"
	"github.com/seventv/api/rest/rest"
	"github.com/seventv/api/rest/v2/routes/auth"
	"github.com/seventv/api/rest/v2/routes/cosmetics"
	"github.com/seventv/api/rest/v2/routes/emotes"
	"github.com/seventv/api/rest/v2/routes/user"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/v2",
		Method: rest.GET,
		Children: []rest.Route{
			auth.New(r.Ctx),
			user.New(r.Ctx),
			emotes.New(r.Ctx),
			cosmetics.New(r.Ctx),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrUnknownRoute()
}
