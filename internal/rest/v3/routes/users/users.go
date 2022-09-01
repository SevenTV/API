package users

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/middleware"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
)

type userRoute struct {
	Ctx global.Context
}

func newUser(gctx global.Context) rest.Route {
	return &userRoute{gctx}
}

func (r *userRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/{user.id}",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 300, []string{"s-maxage=600"}),
		},
	}
}

// Get User
// @Summary Get User
// @Description Get user by ID
// @Param userID path string true "ID of the user"
// @Tags users
// @Produce json
// @Success 200 {object} model.UserModel
// @Router /users/{user.id} [get]
func (r *userRoute) Handler(ctx *rest.Ctx) rest.APIError {
	userID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(userID)
	if err != nil {
		return errors.From(err)
	}

	return ctx.JSON(rest.OK, r.Ctx.Inst().Modelizer.User(user))
}
