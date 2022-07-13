package loaders

import (
	"context"
	"time"

	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteLoader(ctx context.Context, x inst, key string) EmoteLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, structures.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([]structures.Emote, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch emote data from the database
			items := make([]structures.Emote, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted emotes in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedEmote
			}

			// Fetch emotes
			emotes, err := x.query.Emotes(ctx, bson.M{
				key: bson.M{"$in": keys},
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

func batchEmoteLoader(ctx context.Context, x inst, key string) BatchEmoteLoaderByID {
	return dataloader.New(dataloader.Config[primitive.ObjectID, []structures.Emote]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []primitive.ObjectID) ([][]structures.Emote, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Fetch emote data from the database
			items := make([][]structures.Emote, len(keys))
			errs := make([]error, len(keys))

			emotes, err := x.query.Emotes(ctx, bson.M{
				key:                        bson.M{"$in": keys},
				"versions.state.lifecycle": structures.EmoteLifecycleLive,
			}).Items()

			emoteMap := make(map[primitive.ObjectID][]structures.Emote)

			if err == nil {
				for _, e := range emotes {
					s, ok := emoteMap[e.OwnerID]
					if !ok {
						s = []structures.Emote{}
						emoteMap[e.OwnerID] = s
					}

					s = append(s, e)
					emoteMap[e.OwnerID] = s
				}

				for i, v := range keys {
					if x, ok := emoteMap[v]; ok {
						items[i] = x
					}
				}
			} else {
				for i := range errs {
					errs[i] = err
				}
			}

			return items, errs
		},
	})
}
