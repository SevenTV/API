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

func presenceLoader(ctx context.Context, x inst) PresenceLoaderByActorID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []structures.UserPresence[bson.Raw]]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]structures.UserPresence[bson.Raw], []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch presence data from the database
			items := make([][]structures.UserPresence[bson.Raw], len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted presences in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = []structures.UserPresence[bson.Raw]{}
			}

			// Fetch presences
			cur, err := x.mongo.Collection(mongo.CollectionNameUserPresences).Find(ctx, bson.M{
				"actor_id": bson.M{"$in": keys},
			})

			presences := make([]structures.UserPresence[bson.Raw], 0)

			presenceMap := make(map[primitive.ObjectID][]structures.UserPresence[bson.Raw])

			if err := cur.All(ctx, &presences); err != nil {
				return items, errs
			}

			if err == nil {
				for _, p := range presences {
					s, ok := presenceMap[p.UserID]
					if !ok {
						s = []structures.UserPresence[bson.Raw]{}
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
