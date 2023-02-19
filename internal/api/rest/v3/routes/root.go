package routes

import (
	"strconv"
	"time"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/api/rest/v3/routes/auth"
	"github.com/seventv/api/internal/api/rest/v3/routes/config"
	"github.com/seventv/api/internal/api/rest/v3/routes/docs"
	emote_sets "github.com/seventv/api/internal/api/rest/v3/routes/emote-sets"
	"github.com/seventv/api/internal/api/rest/v3/routes/emotes"
	"github.com/seventv/api/internal/api/rest/v3/routes/users"
	"github.com/seventv/api/internal/global"
)

var uptime = time.Now()

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
			config.New(r.Ctx),
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
	return ctx.JSON(rest.OK, HealthResponse{
		Online: true,
		Uptime: strconv.Itoa(int(uptime.UnixMilli())),
	})
}

type HealthResponse struct {
	Online bool   `json:"online"`
	Uptime string `json:"uptime"`
}
