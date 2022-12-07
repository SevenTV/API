package mutation

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v2/gen/generated"
	"github.com/seventv/api/internal/api/gql/v2/gen/model"
	"github.com/seventv/api/internal/api/gql/v2/helpers"
	"github.com/seventv/api/internal/api/gql/v2/types"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

// EditUser implements generated.MutationResolver
func (r *Resolver) EditUser(ctx context.Context, inp model.UserInput, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	targetID, err := primitive.ObjectIDFromHex(inp.ID)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	target, err := r.Ctx.Inst().Loaders.UserByID().Load(targetID)
	if err != nil {
		return nil, err
	}

	if !actor.HasPermission(structures.RolePermissionManageUsers) {
		if target.ID != actor.ID {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("You cannot edit this user")
		}
	}

	if inp.CosmeticPaint != nil {
		paintID, err := primitive.ObjectIDFromHex(*inp.CosmeticPaint)
		if err != nil {
			return nil, errors.ErrBadObjectID()
		}

		// Set the user's paint
		if !paintID.IsZero() {
			res, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).UpdateOne(ctx, bson.M{
				"kind":     "PAINT",
				"data.ref": paintID,
				"user_id":  targetID,
			}, bson.M{"$set": bson.M{"data.selected": true}})
			if err == mongo.ErrNoDocuments || res.ModifiedCount == 0 {
				return nil, errors.ErrInsufficientPrivilege().SetDetail("You do not own this paint")
			} else if err != nil {
				zap.S().Errorw("mongo, failed to select entitlement", "error", err)
			}
		}

		// Disable other paints
		if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).UpdateMany(ctx, bson.M{
			"kind":     "PAINT",
			"data.ref": bson.M{"$not": bson.M{"$eq": paintID}},
			"user_id":  targetID,
		}, bson.M{"$set": bson.M{"data.selected": false}}); err != nil {
			zap.S().Errorw("mongo, failed to update other entitlements", "error", err)
			return nil, err
		}
	}

	if inp.CosmeticBadge != nil {
		badgeID, err := primitive.ObjectIDFromHex(*inp.CosmeticBadge)
		if err != nil {
			return nil, errors.ErrBadObjectID()
		}

		// Set the user's badge
		if !badgeID.IsZero() {
			res, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).UpdateOne(ctx, bson.M{
				"kind":     "BADGE",
				"data.ref": badgeID,
				"user_id":  targetID,
			}, bson.M{"$set": bson.M{"data.selected": true}})
			if err == mongo.ErrNoDocuments || res.ModifiedCount == 0 {
				return nil, errors.ErrInsufficientPrivilege().SetDetail("You do not own this badge")
			} else if err != nil {
				zap.S().Errorw("mongo, failed to select entitlement", "error", err)
			}
		}

		// Disable other badges
		if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).UpdateMany(ctx, bson.M{
			"kind":     "BADGE",
			"data.ref": bson.M{"$not": bson.M{"$eq": badgeID}},
			"user_id":  targetID,
		}, bson.M{"$set": bson.M{"data.selected": false}}); err != nil {
			zap.S().Errorw("mongo, failed to update other entitlements", "error", err)
		}
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func New(r types.Resolver) generated.MutationResolver {
	return &Resolver{
		Resolver: r,
	}
}
