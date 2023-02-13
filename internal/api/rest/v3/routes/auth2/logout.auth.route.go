package auth

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/svc/auth"
	"github.com/seventv/common/errors"
)

type logoutRoute struct {
	gctx global.Context
}

func newLogout(gctx global.Context) rest.Route {
	return &logoutRoute{gctx}
}

func (r *logoutRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/logout",
		Method: rest.GET,
	}
}

func (r *logoutRoute) Handler(ctx *rest.Ctx) errors.APIError {
	cookie := r.gctx.Inst().Auth.Cookie(auth.COOKIE_AUTH, "", 0)

	ctx.Response.Header.SetCookie(cookie)

	return nil
}
