package user

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Cosmetics implements generated.UserResolver
func (r *Resolver) Cosmetics(ctx context.Context, obj *model.User) ([]*model.UserCosmetic, error) {
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEntitlements).Find(ctx, bson.M{
		"user_id": obj.ID,
		"kind":    bson.M{"$in": bson.A{structures.CosmeticKindBadge, structures.CosmeticKindNametagPaint}},
	}, options.Find().SetProjection(bson.M{"data.ref": 1, "data.selected": 1, "kind": 1}))
	if err != nil {
		return nil, errors.ErrInternalServerError()
	}

	ents := []structures.Entitlement[structures.EntitlementDataBaseSelectable]{}

	if err = cur.All(ctx, &ents); err != nil {
		return nil, errors.ErrInternalServerError()
	}

	result := make([]*model.UserCosmetic, len(ents))
	for i, ent := range ents {
		result[i] = &model.UserCosmetic{
			ID:       ent.Data.RefID,
			Selected: ent.Data.Selected,
			Kind:     model.CosmeticKind(ent.Kind),
		}
	}

	return result, nil
}
