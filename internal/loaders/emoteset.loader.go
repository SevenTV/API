package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/instance"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteSetByID(gCtx global.Context) instance.EmoteSetLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, structures.EmoteSet]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([]structures.EmoteSet, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch emote set data from the database
			models := make([]structures.EmoteSet, len(keys))
			errs := make([]error, len(keys))

			sets, err := gCtx.Inst().Query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": keys}}).Items()

			m := make(map[primitive.ObjectID]structures.EmoteSet)
			if err == nil {
				for _, set := range sets {
					m[set.ID] = set
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						models[i] = x
					} else {
						errs[i] = mongo.ErrNoDocuments
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

func emoteSetByUserID(gCtx global.Context) instance.BatchEmoteSetLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []structures.EmoteSet]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]structures.EmoteSet, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch emote sets
			modelLists := make([][]structures.EmoteSet, len(keys))
			errs := make([]error, len(keys))

			sets, err := gCtx.Inst().Query.UserEmoteSets(ctx, bson.M{"owner_id": bson.M{"$in": keys}})

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
