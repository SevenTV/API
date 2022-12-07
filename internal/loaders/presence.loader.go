package loaders

import (
	"context"
	"time"

	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func presenceLoader[T structures.UserPresenceData](ctx context.Context, x inst, kind structures.UserPresenceKind, key string) *dataloader.DataLoader[primitive.ObjectID, []structures.UserPresence[T]] {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []structures.UserPresence[T]]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]structures.UserPresence[T], []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch presence data from the database
			items := make([][]structures.UserPresence[T], len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted presences in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = []structures.UserPresence[T]{}
			}

			// Fetch presences
			f := bson.M{key: bson.M{"$in": keys}}

			if kind > 0 {
				f["kind"] = kind
			}

			cur, err := x.mongo.Collection(mongo.CollectionNameUserPresences).Find(ctx, f)

			presences := make([]structures.UserPresence[T], 0)

			presenceMap := make(map[primitive.ObjectID][]structures.UserPresence[T])

			if err := cur.All(ctx, &presences); err != nil {
				return items, errs
			}

			if err == nil {
				for _, p := range presences {
					s, ok := presenceMap[p.UserID]
					if !ok {
						s = []structures.UserPresence[T]{}
						presenceMap[p.UserID] = s
					}

					s = append(s, p)
					presenceMap[p.UserID] = s
				}

				for i, v := range keys {
					if x, ok := presenceMap[v]; ok {
						items[i] = x
					}
				}
			}

			return items, errs
		},
	})
}
