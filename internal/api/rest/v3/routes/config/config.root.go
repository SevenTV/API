package config

import (
	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
)

type Route struct {
	Ctx global.Context
}

func New(gctx global.Context) rest.Route {
	return &Route{gctx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/config/{name}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 60, nil),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	// get path
	name, ok := ctx.UserValue("name").String()
	if !ok {
		return errors.ErrInvalidRequest().SetDetail("Missing config name")
	}

	sys, err := r.Ctx.Inst().Mongo.System(ctx)
	if err != nil {
		ctx.Log().Errorw("failed to get system config",
			"err", err,
		)

		return errors.ErrInternalServerError()
	}

	var t any

	switch name {
	case "extension":
		t = sys.Config.Extension
	case "extension-beta":
		t = sys.Config.ExtensionBeta
	default:
		return errors.ErrInvalidRequest().SetDetail("Invalid config name")
	}

	return ctx.JSON(rest.OK, t)
}
