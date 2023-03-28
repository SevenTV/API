package cosmetics

import (
	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
)

type avatars struct {
	Ctx global.Context
}

func newAvatars(gCtx global.Context) rest.Route {
	return &avatars{gCtx}
}

// Config implements rest.Route
func (r *avatars) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/avatars",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 86400, []string{"public"}),
		},
	}
}

// Handler implements rest.Route
func (r *avatars) Handler(ctx *rest.Ctx) errors.APIError {
	return errors.ErrEndOfLife().SetDetail("This endpoint is no longer available. Please use the EventAPI or the Get User endpoint instead.")
}
