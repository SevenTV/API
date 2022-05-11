package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/global"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func userEmotesLoader(gCtx global.Context) UserEmotesLoader {
	return dataloader.New(dataloader.Config[string, []structures.ActiveEmote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []string) ([][]structures.ActiveEmote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			modelLists := make([][]structures.ActiveEmote, len(keys))
			errs := make([]error, len(keys))

			ids := make([]primitive.ObjectID, len(keys))
			for i, k := range keys {
				ids[i], _ = primitive.ObjectIDFromHex(k)
			}

			sets, err := gCtx.Inst().Query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": ids}}).Items()
			if err == nil {
				m := make(map[primitive.ObjectID][]structures.ActiveEmote)
				// iterate over sets
				for _, set := range sets {
					// iterate over emotes of set
					for _, ae := range set.Emotes {
						if ae.Emote == nil {
							continue
						}

						m[set.ID] = append(m[set.ID], ae)
					}
				}

				for i, v := range ids {
					if x, ok := m[v]; ok {
						modelLists[i] = x
					}
				}
			}

			return modelLists, errs
		},
	})
}
