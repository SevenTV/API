package routes

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/api/internal/rest/v2/routes/auth"
	"github.com/seventv/api/internal/rest/v2/routes/chatterino"
	"github.com/seventv/api/internal/rest/v2/routes/cosmetics"
	"github.com/seventv/api/internal/rest/v2/routes/downloads"
	"github.com/seventv/api/internal/rest/v2/routes/emotes"
	"github.com/seventv/api/internal/rest/v2/routes/user"
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
		URI:    "/v2" + r.Ctx.Config().Http.VersionSuffix,
		Method: rest.GET,
		Children: []rest.Route{
			auth.New(r.Ctx),
			user.New(r.Ctx),
			emotes.New(r.Ctx),
			cosmetics.New(r.Ctx),
			downloads.New(r.Ctx),
			chatterino.New(r.Ctx),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrUnknownRoute()
}
