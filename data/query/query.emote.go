package query

import (
	"context"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

func (q *Query) Emotes(ctx context.Context, filter bson.M) *QueryResult[structures.Emote] {
	qr := QueryResult[structures.Emote]{}
	items := []structures.Emote{}

	bans, err := q.Bans(ctx, BanQueryOptions{
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectNoOwnership | structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return qr.setError(err)
	}

	if len(bans.NoOwnership) > 0 {
		filter["owner_id"] = bson.M{"$not": bson.M{"$in": bans.NoOwnership.KeySlice()}}
	}

	cur, err := q.mongo.Collection(mongo.CollectionNameEmotes).Find(ctx, filter)
	if err != nil {
		return qr.setError(err)
	}

	if err := cur.All(ctx, &items); err != nil {
		return qr.setError(err)
	}

	return qr.setItems(items)
}
