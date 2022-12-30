package query

import (
	"context"
	"time"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (q *Query) Emotes(ctx context.Context, filter bson.M) *QueryResult[structures.Emote] {
	qr := QueryResult[structures.Emote]{}

	bans, err := q.Bans(ctx, BanQueryOptions{
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return qr.setError(err)
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Aggregate(ctx, mongo.Pipeline{
		{{
			Key:   "$match",
			Value: bson.M{"owner_id": bson.M{"$not": bson.M{"$in": bans.NoOwnership.KeySlice()}}},
		}},
		{{
			Key:   "$match",
			Value: filter,
		}},
	}, options.MergeAggregateOptions().SetBatchSize(25).SetMaxAwaitTime(time.Second*30))
	if err != nil {
		zap.S().Errorw("failed to create query to aggregate emotes", "error", err)

		return qr.setError(err)
	}

	items := []structures.Emote{}

	if err := cur.All(ctx, &items); err != nil {
		zap.S().Errorw("failed to decode emotes", "error", err)

		return qr.setError(err)
	}

	owners, err := q.Users(ctx, bson.M{"_id": bson.M{
		"$in": utils.Map(items, func(x structures.Emote) primitive.ObjectID {
			return x.OwnerID
		}),
	}}).Items()
	if err != nil {
		zap.S().Errorw("failed to fetch emote owners", "error", err)

		return qr.setError(err)
	}

	ownerMap := map[primitive.ObjectID]structures.User{}
	for _, u := range owners {
		ownerMap[u.ID] = u
	}

	for i := range items {
		if owner, ok := ownerMap[items[i].OwnerID]; ok {
			items[i].Owner = &owner
		}
	}

	return qr.setItems(items)
}
