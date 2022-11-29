package loaders

import (
	"context"
	"time"

	"github.com/seventv/api/data/query"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteSetByID(ctx context.Context, x inst) EmoteSetLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, structures.EmoteSet]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([]structures.EmoteSet, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch emote set data from the database
			models := make([]structures.EmoteSet, len(keys))
			errs := make([]error, len(keys))

			result := x.query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": keys}}, query.QueryEmoteSetsOptions{
				FetchOrigins: true,
			})
			if result.Empty() {
				return models, errs
			}
			sets, err := result.Items()

			m := make(map[primitive.ObjectID]structures.EmoteSet)
			if err == nil {
				for _, set := range sets {
					m[set.ID] = set
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						models[i] = x
					} else {
						errs[i] = errors.ErrUnknownEmoteSet()
					}
				}
			} else {
				for i := range errs {
					errs[i] = err
				}
			}

			return models, errs
		},
	})
}

func emoteSetByUserID(ctx context.Context, x inst) BatchEmoteSetLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []structures.EmoteSet]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]structures.EmoteSet, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch emote sets
			modelLists := make([][]structures.EmoteSet, len(keys))
			errs := make([]error, len(keys))

			sets, err := x.query.UserEmoteSets(ctx, bson.M{"owner_id": bson.M{"$in": keys}})

			if err == nil {
				for i, v := range keys {
					if x, ok := sets[v]; ok {
						modelLists[i] = x
					}
				}
			} else {
				for i := range errs {
					errs[i] = err
				}
			}

			return modelLists, errs
		},
	})
}
