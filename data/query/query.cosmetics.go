package query

import (
	"context"
	"time"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongod "go.mongodb.org/mongo-driver/mongo"
)

func (q *Query) Cosmetics(ctx context.Context, ids utils.Set[primitive.ObjectID]) ([]structures.Cosmetic[bson.Raw], error) {
	mtx := q.mtx("ManyCosmetics")
	mtx.Lock()
	defer mtx.Unlock()

	k := q.key("cosmetics")

	var (
		result = []structures.Cosmetic[bson.Raw]{}
		err    error
		cur    *mongod.Cursor
	)

	// Get cached
	if ok := q.getFromMemCache(ctx, k, &result); ok {
		goto end
	}

	// Query
	cur, err = q.mongo.Collection(mongo.CollectionNameCosmetics).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	if err = cur.All(ctx, &result); err != nil {
		return nil, err
	}

	// Set cache
	if err = q.setInMemCache(ctx, k, &result, time.Second*30); err != nil {
		return nil, err
	}

end:
	return utils.Filter(result, func(x structures.Cosmetic[bson.Raw]) bool {
		return ids.Has(x.ID)
	}), nil
}
