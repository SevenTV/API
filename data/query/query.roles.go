package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (q *Query) Roles(ctx context.Context, filter bson.M) ([]structures.Role, error) {
	mtx := q.mtx("ManyRoles")
	mtx.Lock()
	defer mtx.Unlock()

	hs := "all"
	if len(filter) > 0 {
		f, _ := json.Marshal(filter)
		h := sha256.New()
		h.Write(f)
		hs = hex.EncodeToString(h.Sum((nil)))
	}
	k := q.key(fmt.Sprintf("roles:%s", hs))
	result := []structures.Role{}

	// Get cached
	if ok := q.getFromMemCache(ctx, k, &result); ok {
		return result, nil
	}

	// Query
	cur, err := q.mongo.Collection(mongo.CollectionNameRoles).Find(ctx, filter, options.Find().SetSort(bson.M{"position": -1}))
	if err == nil {
		if err = cur.All(ctx, &result); err != nil {
			return nil, err
		}
	}

	// Set cache
	if err = q.setInMemCache(ctx, k, &result, time.Second*10); err != nil {
		return nil, err
	}
	return result, nil
}

type ManyRolesOptions struct {
	DefaultOnly bool
}
