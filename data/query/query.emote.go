package query

import (
	"context"

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

	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Find(ctx, filter, options.Find().SetNoCursorTimeout(true).SetBatchSize(10))
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
