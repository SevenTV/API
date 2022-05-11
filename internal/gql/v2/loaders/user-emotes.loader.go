package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func userEmotesLoader(gCtx global.Context) *dataloader.DataLoader[string, []*model.Emote] {
	return dataloader.New(dataloader.Config[string, []*model.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []string) ([][]*model.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			modelLists := make([][]*model.Emote, len(keys))
			errs := make([]error, len(keys))

			ids := make([]primitive.ObjectID, len(keys))
			for i, k := range keys {
				ids[i], _ = primitive.ObjectIDFromHex(k)
			}

			sets, err := gCtx.Inst().Query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": ids}}).Items()
			if err == nil {
				m := make(map[primitive.ObjectID][]*model.Emote)
				// iterate over sets
				for _, set := range sets {
					// iterate over emotes of set
					for _, ae := range set.Emotes {
						if ae.Emote == nil {
							continue
						}
						em := helpers.EmoteStructureToModel(gCtx, *ae.Emote)

						// set "alias"?
						if ae.Name != em.Name {
							em.OriginalName = &ae.Emote.Name
							em.Name = ae.Name
						}

						m[set.ID] = append(m[set.ID], em)
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
