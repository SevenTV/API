package users

import (
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/api/rest/middleware"
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
		Middleware: []rest.Middleware{
			middleware.Auth(r.gctx, true),
		},
	}
}

func (r *userDeleteRoute) Handler(ctx *rest.Ctx) rest.APIError {
	// make sure actor has permission to delete users
	actor, ok := ctx.GetActor()
	if !ok || !actor.HasPermission(structures.RolePermissionManageUsers) {
		return errors.ErrInsufficientPrivilege()
	}

	victimID, err := ctx.UserValue("user.id").ObjectID()
	if err != nil {
		return errors.From(err)
	}

	victim, err := r.gctx.Inst().Query.Users(ctx, bson.M{
		"_id": victimID,
	}).First()
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return errors.ErrUnknownUser()
		}

		return errors.From(err)
	}

	res := userDeleteResponse{}
	// delete user
	if res.DocumentDeletedCount, err = r.gctx.Inst().Mutate.DeleteUser(ctx, mutate.DeleteUserOptions{
		Actor:  actor,
		Victim: victim,
	}); err != nil {
		return errors.From(err)
	}

	// Create audit log
	log := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetKind(structures.AuditLogKindDeleteUser).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmoteSet).
		SetTargetID(victimID)

	if _, err = r.gctx.Inst().Mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, log.AuditLog); err != nil {
		ctx.Log().Errorw("mongo, failed to write audit log entry for deleted user")
	}

	ctx.Log().Infow("user deleted", "victim_id", victimID)

	return ctx.JSON(rest.OK, res)
}

type userDeleteResponse struct {
	DocumentDeletedCount int `json:"document_deleted_count"`
}
