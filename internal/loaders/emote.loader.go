package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/instance"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteByID(gCtx global.Context) instance.EmoteLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, structures.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([]structures.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch emote data from the database
			items := make([]structures.Emote, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted emotes in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedEmote
			}

			// Fetch emotes
			emotes, err := gCtx.Inst().Query.Emotes(ctx, bson.M{
				"versions.id": bson.M{"$in": keys},
			}).Items()

			if err == nil {
				m := make(map[primitive.ObjectID]structures.Emote)
				for _, e := range emotes {
					for _, ver := range e.Versions {
						m[ver.ID] = e
					}
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						ver, _ := x.GetVersion(v)
						if ver.ID.IsZero() || ver.IsUnavailable() {
							continue
						}
						x.ID = v
						items[i] = x
					}
				}
			}

			return items, errs
		},
	})
}
