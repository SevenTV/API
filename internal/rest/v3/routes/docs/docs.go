package docs

import (
	"os"

	"github.com/seventv/api/internal/global"
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
		URI:    "/docs",
		Method: rest.GET,
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	b, err := os.ReadFile("docs/v3/swagger.json")
	if err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	ctx.SetBody(b)
	return nil
}
