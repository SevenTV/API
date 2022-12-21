package query

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Cosmetics implements generated.QueryResolver
func (r *Resolver) Cosmetics(ctx context.Context, list []primitive.ObjectID) (*model.CosmeticsQuery, error) {
	result := model.CosmeticsQuery{}

	filter := bson.M{}
	if len(list) > 0 {
		filter["_id"] = bson.M{"$in": list}
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).Find(ctx, filter)
	if err != nil {
		return &result, errors.ErrInternalServerError()
	}

	cosmetics := []structures.Cosmetic[bson.Raw]{}
	if err := cur.All(ctx, &cosmetics); err != nil {
		return nil, errors.ErrInternalServerError()
	}

	paints := []*model.CosmeticPaint{}
	badges := []*model.CosmeticBadge{}

	for _, cosmetic := range cosmetics {
		switch cosmetic.Kind {
		case structures.CosmeticKindNametagPaint:
			c, err := structures.ConvertCosmetic[structures.CosmeticDataPaint](cosmetic)
			if err == nil {
				paints = append(paints, modelgql.CosmeticPaint(r.Ctx.Inst().Modelizer.Paint(c)))
			}
		case structures.CosmeticKindBadge:
			c, err := structures.ConvertCosmetic[structures.CosmeticDataBadge](cosmetic)
			if err == nil {
				badges = append(badges, modelgql.CosmeticBadge(r.Ctx.Inst().Modelizer.Badge(c)))
			}
		}
	}

	return &model.CosmeticsQuery{
		Paints: paints,
		Badges: badges,
	}, nil
}
