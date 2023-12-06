package users

import (
	"encoding/json"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
)

type userUpdateConnectionRoute struct {
	gctx global.Context
}

func newUserMergeRoute(gctx global.Context) *userUpdateConnectionRoute {
	return &userUpdateConnectionRoute{gctx}
}

func (r *userUpdateConnectionRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/{user.id}/connections/{user-connection.id}",
		Method: rest.PATCH,
		Middleware: []rest.Middleware{
			middleware.Auth(r.gctx, true),
		},
	}
}

func (r *userUpdateConnectionRoute) Handler(ctx *rest.Ctx) rest.APIError {
	// make sure actor has permission to delete users
	actor, ok := ctx.GetActor()
	if !ok || !actor.HasPermission(structures.RolePermissionManageUsers) {
		return errors.ErrInsufficientPrivilege()
	}

	userID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	connectionID, _ := ctx.UserValue("user-connection.id").String()

	var body updateUserConnections
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		return errors.ErrInvalidRequest()
	}

	target, err := r.gctx.Inst().Query.Users(ctx, bson.M{
		"_id": userID,
	}).First()
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return errors.ErrUnknownUser().SetDetail("target")
		}

		return errors.From(err)
	}

	if !body.NewUserID.IsZero() {
		victim, err := r.gctx.Inst().Query.Users(ctx, bson.M{
			"_id": body.NewUserID,
		}).First()
		if err != nil {
			if errors.Compare(err, errors.ErrNoItems()) {
				return errors.ErrUnknownUser().SetDetail("victim")
			}

			return errors.From(err)
		}

		if err = r.gctx.Inst().Mutate.TransferUserConnection(ctx, target, victim, connectionID); err != nil {
			return errors.From(err)
		}
	}

	// TODO: add mutation to audit log

	return ctx.JSON(rest.OK, struct{}{})
}

type updateUserConnections struct {
	NewUserID primitive.ObjectID `json:"new_user_id"`
}
