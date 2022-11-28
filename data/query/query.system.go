package query

import (
	"context"
	"time"

	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

func (q *Query) GlobalEmoteSet(ctx context.Context) (structures.EmoteSet, error) {
	mtx := q.mtx("GlobalEmoteSet")
	mtx.Lock()
	defer mtx.Unlock()

	k := q.key("global_emote_set")

	set := structures.EmoteSet{}

	// Get cached
	if ok := q.getFromMemCache(ctx, k, &set); ok {
		return set, nil
	}

	var err error
	sys, err := q.mongo.System(ctx)
	if err != nil {
		return set, err
	}
	set, err = q.EmoteSets(ctx, bson.M{"_id": sys.EmoteSetID}).First()
	if err != nil {
		return set, err
	}

	// Set cache
	if err := q.setInMemCache(ctx, k, set, time.Second*30); err != nil {
		return set, err
	}

	return set, nil
}
