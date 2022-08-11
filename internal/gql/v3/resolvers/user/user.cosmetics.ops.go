package user

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *ResolverOps) Cosmetics(ctx context.Context, obj *model.UserOps, id primitive.ObjectID, selectedArg *bool) (*bool, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	if actor.ID != obj.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	// Get the cosmetic item
	cos := structures.Cosmetic[bson.Raw]{}
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).FindOne(ctx, bson.M{
		"_id": id,
	}).Decode(&cos); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownCosmetic()
		}

		r.Z().Errorw("failed to get cosmetic", "error", err)
		return nil, errors.ErrInternalServerError()
	}

	selected := true
	if selectedArg != nil {
		selected = *selectedArg
	}

	// Set the cosmetic's selection state
	w := []mongo.WriteModel{}

	w = append(w, &mongo.UpdateOneModel{
		Filter: bson.M{
			"kind":     cos.Kind,
			"data.ref": cos.ID,
			"user_id":  obj.ID,
		},
		Update: bson.M{"$set": bson.M{"data.selected": selected}},
	})

	w = append(w, &mongo.UpdateManyModel{
		Filter: bson.M{
			"kind":     cos.Kind,
			"data.ref": bson.M{"$not": bson.M{"$eq": cos.ID}},
			"user_id":  obj.ID,
		},
		Update: bson.M{
			"$set": bson.M{"data.selected": false},
		},
	})

	if _, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).BulkWrite(ctx, w); err != nil {
		r.Z().Errorw("failed to update user cosmetic state", "error", err)

		return nil, errors.ErrInternalServerError()
	}

	return &selected, nil
}
