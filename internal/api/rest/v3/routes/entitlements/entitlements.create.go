package entitlements

import (
	"encoding/json"
	"time"

	"github.com/seventv/api/internal/api/rest/middleware"
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type create struct {
	gctx global.Context
}

func newCreate(gctx global.Context) rest.Route {
	return &create{gctx}
}

func (r *create) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "",
		Method: rest.POST,
		Middleware: []rest.Middleware{
			middleware.Auth(r.gctx, true),
		},
	}
}

func (r *create) Handler(ctx *rest.Ctx) rest.APIError {
	actor, ok := ctx.GetActor()
	if !ok || !actor.HasPermission(structures.RolePermissionManageEntitlements) {
		return errors.ErrInsufficientPrivilege()
	}

	var body createEntitlementData
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		return errors.ErrInvalidRequest()
	}

	// Validate ROLE entitlements
	// Needs superadmin
	if body.Kind == structures.EntitlementKindRole && !actor.HasPermission(structures.RolePermissionSuperAdministrator) {
		return errors.ErrInsufficientPrivilege()
	}

	id := primitive.NewObjectIDFromTimestamp(time.Now())

	eb := structures.NewEntitlementBuilder(structures.Entitlement[structures.EntitlementDataBase]{
		ID: id,
	}).
		SetKind(body.Kind).
		SetUserID(body.UserID).
		SetCondition(body.Condition).
		SetData(structures.EntitlementDataBase{
			RefID: body.ObjectRef,
		}).
		SetApp(structures.EntitlementApp{
			Name:    body.AppName,
			ActorID: actor.ID.Hex(),
			State:   body.AppState,
		})

	// Set claim (only if UserID empty)
	if body.Claim != nil && body.UserID.IsZero() {
		u, _ := r.gctx.Inst().Loaders.UserByConnectionID(body.Claim.Platform).Load(body.Claim.ID)

		if u.ID.IsZero() {
			eb.SetClaim(structures.EntitlementClaim{
				Platform: body.Claim.Platform,
				ID:       body.Claim.ID,
			})
		} else {
			// if the user was found by claim, assign the entitlement directly
			eb.SetUserID(u.ID)
		}
	}

	if _, err := r.gctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).InsertOne(ctx, eb.Entitlement); err != nil {
		ctx.Log().Errorw("mongo, couldn't create entitlement")

		return errors.ErrInternalServerError()
	}

	return ctx.JSON(rest.OK, eb.Entitlement)
}

type createEntitlementData struct {
	Kind      structures.EntitlementKind      `json:"kind"`
	ObjectRef primitive.ObjectID              `json:"object_ref"`
	UserID    primitive.ObjectID              `json:"user_id"`
	Condition structures.EntitlementCondition `json:"condition"`
	Claim     *structures.EntitlementClaim    `json:"claim"`
	AppName   string                          `json:"app_name"`
	AppState  map[string]any                  `json:"app_state"`
}
