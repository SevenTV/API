package users

import (
	"encoding/json"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
)

type userMergeRoute struct {
	gctx global.Context
}

func newUserMergeRoute(gctx global.Context) *userMergeRoute {
	return &userMergeRoute{gctx}
}

func (r *userMergeRoute) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/merge",
		Method: rest.POST,
	}
}

func (r *userMergeRoute) Handler(ctx *rest.Ctx) rest.APIError {
	// make sure actor has permission to delete users
	actor, ok := ctx.GetActor()
	if !ok || !actor.HasPermission(structures.RolePermissionManageUsers) {
		return errors.ErrUnauthorized()
	}

	var body userMergeBody
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		return errors.ErrInvalidRequest()
	}

	donorID, err := primitive.ObjectIDFromHex(body.DonorUserID)
	if err != nil {
		return errors.ErrInvalidRequest()
	}
	recipientID, err := primitive.ObjectIDFromHex(body.RecipientUserID)
	if err != nil {
		return errors.ErrInvalidRequest()
	}

	// delete user
	if err = r.gctx.Inst().Mutate.MergeUserConnections(ctx, donorID, recipientID, body.ConnectionID); err != nil {
		return errors.ErrInternalServerError()
	}

	// TODO: add mutation to audit log

	return ctx.JSON(rest.OK, struct{}{})
}

type userMergeBody struct {
	RecipientUserID string `json:"recipient_user_id"`
	DonorUserID     string `json:"donor_user_id"`
	ConnectionID    string `json:"connection_id"`
}
