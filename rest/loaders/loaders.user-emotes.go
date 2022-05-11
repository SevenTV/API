package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/global"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emotesByEmoteSetID(gCtx global.Context) *BatchEmoteLoader {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []*structures.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]*structures.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			modelLists := make([][]*structures.Emote, len(keys))
			errs := make([]error, len(keys))

			sets, err := gCtx.Inst().Query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": keys}}).Items()
			if err == nil {
				m := make(map[primitive.ObjectID][]*structures.Emote)
				// iterate over sets
				for _, set := range sets {
					// iterate over emotes of set
					for _, ae := range set.Emotes {
						// set "alias"?
						if ae.Name != ae.Emote.Name {
							ae.Emote.Name = ae.Name
						}

						m[set.ID] = append(m[set.ID], ae.Emote)
					}
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						modelLists[i] = x
					}
				}
			}

			return modelLists, errs
		},
	})
}
