package query

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

// Cosmetics implements generated.QueryResolver
func (r *Resolver) Cosmetics(ctx context.Context) (*model.CosmeticsQuery, error) {
	result := model.CosmeticsQuery{}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameCosmetics).Find(ctx, bson.M{})
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
				paints = append(paints, helpers.CosmeticPaintStructureToModel(c))
			}
		case structures.CosmeticKindBadge:
			c, err := structures.ConvertCosmetic[structures.CosmeticDataBadge](cosmetic)
			if err == nil {
				badges = append(badges, helpers.CosmeticBadgeStructureToModel(c))
			}
		}
	}

	return &model.CosmeticsQuery{
		Paints: paints,
		Badges: badges,
	}, nil
}
