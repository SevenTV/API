package user

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ResolverPartial struct {
	types.Resolver
}

func NewPartial(r types.Resolver) generated.UserPartialResolver {
	return &ResolverPartial{r}
}

func (r *ResolverPartial) EmoteSets(ctx context.Context, obj *model.UserPartial) ([]*model.EmoteSetPartial, error) {
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmoteSets).Find(ctx, bson.M{
		"owner_id": obj.ID,
	}, options.Find().SetProjection(bson.M{
		"_id":      1,
		"name":     1,
		"capacity": 1,
	}))
	if err != nil {
		return nil, err
	}

	result := []*model.EmoteSetPartial{}

	for cur.Next(ctx) {
		set := structures.EmoteSet{}

		if err := cur.Decode(&set); err != nil {
			continue
		}

		result = append(result, &model.EmoteSetPartial{
			ID:       set.ID,
			Name:     set.Name,
			Capacity: int(set.Capacity),
		})
	}

	return result, nil
}
