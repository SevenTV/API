package users

import (
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
)

type userDeleteRoute struct {
	gctx global.Context
}

func newUserDeleteRoute(gctx global.Context) *userDeleteRoute {
	return &userDeleteRoute{gctx}
}

func (r *userDeleteRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/{user.id}",
		Method: rest.DELETE,
	}
}

func (r *userDeleteRoute) Handler(ctx *rest.Ctx) rest.APIError {
	// make sure actor has permission to delete users
	actor, ok := ctx.GetActor()
	if !ok || !actor.HasPermission(structures.RolePermissionManageUsers) {
		return errors.ErrUnauthorized()
	}

	userID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	res := userDeleteResponse{}
	// delete user
	if res.DocumentDeletedCount, err = r.gctx.Inst().Mutate.DeleteUser(ctx, userID); err != nil {
		return errors.ErrInternalServerError()
	}

	return ctx.JSON(rest.OK, res)
}

type userDeleteResponse struct {
	DocumentDeletedCount int `json:"documentDeletedCount"`
}
