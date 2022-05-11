package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v2/gen/model"
	"github.com/seventv/api/gql/v2/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteByID(gCtx global.Context) *dataloader.DataLoader[string, *model.Emote] {
	return dataloader.New(dataloader.Config[string, *model.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []string) ([]*model.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch emote data from the database
			models := make([]*model.Emote, len(keys))
			errs := make([]error, len(keys))

			// Parse object IDs
			ids := make([]primitive.ObjectID, len(keys))
			for i, k := range keys {
				id, err := primitive.ObjectIDFromHex(k)
				if err != nil {
					errs[i] = err
					continue
				}
				ids[i] = id
			}

			// Fetch emotes
			emotes, err := gCtx.Inst().Query.Emotes(ctx, bson.M{
				"versions.id": bson.M{"$in": ids},
			}).Items()

			if err == nil {
				m := make(map[primitive.ObjectID]structures.Emote)
				for _, e := range emotes {
					for _, ver := range e.Versions {
						m[ver.ID] = e
					}
				}

				for i, v := range ids {
					if x, ok := m[v]; ok {
						ver, _ := x.GetVersion(v)
						if ver.ID.IsZero() || ver.IsUnavailable() {
							continue
						}
						x.ID = v
						models[i] = helpers.EmoteStructureToModel(gCtx, x)
					}
				}
			}

			return models, errs
		},
	})
}
