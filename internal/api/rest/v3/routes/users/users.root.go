package users

import (
	"github.com/seventv/api/internal/api/rest/rest"
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
		URI:    "/users",
		Method: rest.GET,
		Children: []rest.Route{
			newUser(r.Ctx),
			newUserConnection(r.Ctx),
			newPictureUpload(r.Ctx),
			newUserPresenceWriteRoute(r.Ctx),
			newUserDeleteRoute(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

// @Summary Search Users
// @Description Search for users
// @Tags users
// @Produce json
// @Param query query string false "search by username, user id, channel name or channel id"
// @Success 200
// @Router /users [get]
func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return ctx.JSON(rest.OK, struct{}{})
}
