package user

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (r *ResolverOps) Cosmetics(ctx context.Context, obj *model.UserOps, update model.UserCosmeticUpdate) (*bool, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	if actor.ID != obj.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	id := update.ID
	kind := update.Kind
	selected := update.Selected

	w := []mongo.WriteModel{}

	// Get the cosmetic item
	cos := structures.Cosmetic[bson.Raw]{}
	if !id.IsZero() {
		if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).FindOne(ctx, bson.M{
			"_id":  id,
			"kind": kind,
		}).Decode(&cos); err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, errors.ErrUnknownCosmetic()
			}

			r.Z().Errorw("failed to get cosmetic", "error", err)

			return nil, errors.ErrInternalServerError()
		}

		w = append(w, &mongo.UpdateOneModel{
			Filter: bson.M{
				"kind":     kind,
				"data.ref": cos.ID,
				"user_id":  obj.ID,
			},
			Update: bson.M{"$set": bson.M{"data.selected": selected}},
		})
	}

	w = append(w, &mongo.UpdateManyModel{
		Filter: bson.M{
			"kind":     kind,
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

	return utils.PointerOf(true), nil
}
