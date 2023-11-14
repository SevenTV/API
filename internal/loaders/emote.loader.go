package loaders

import (
	"context"
	"encoding/json"
	"time"

	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
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

			// Fetch emotes from cache
			cachedEmotes, err := getEmotesFromCache(ctx, x, keys)
			if err != nil {
				zap.S().Errorw("redis failed to fetch emotes from cache", "error", err)
				errs = append(errs, err)
			}

			remainingKeys := []primitive.ObjectID{}
		keysLoop:
			for i, key := range keys {
				for _, emote := range cachedEmotes {
					if emote.ID == key {
						items[i] = emote
						continue keysLoop
					}
				}

				// key was not found in cache, so we get it from mongo
				remainingKeys = append(remainingKeys, key)
			}

			// Fetch emotes
			emotes, err := x.query.Emotes(ctx, bson.M{
				key: bson.M{"$in": remainingKeys},
			}).Items()

			if err == nil {
				m := make(map[primitive.ObjectID]structures.Emote)
				for _, e := range emotes {
					for _, ver := range e.Versions {
						m[ver.ID] = e
					}
				}

				for i, v := range keys {
					if emote, ok := m[v]; ok {
						ver, _ := emote.GetVersion(v)
						if ver.ID.IsZero() {
							continue
						}

						emote.ID = v
						emote.VersionRef = &ver
						items[i] = emote

						// store emote in redis cache
						err = setEmoteInCache(ctx, x, emote)
						if err != nil {
							zap.S().Errorw("redis failed to set emote in cache", "error", err)
							errs = append(errs, err)
						}
					}
				}
			}

			return items, errs
		},
	})
}

var cacheKeyEmotes = "cache.emotes."

func getEmotesFromCache(ctx context.Context, x inst, baseKeys []primitive.ObjectID) ([]structures.Emote, error) {
	keys := make([]string, len(baseKeys))

	for _, key := range baseKeys {
		keys = append(keys, cacheKeyEmotes+key.String())
	}

	emotes := []structures.Emote{}

	res := x.redis.RawClient().MGet(ctx, keys...)
	if res.Err() != nil {
		return nil, res.Err()
	}

	err := json.Unmarshal([]byte(res.String()), &emotes)
	if err != nil {
		return nil, err
	}

	return emotes, nil
}

func setEmoteInCache(ctx context.Context, x inst, emote structures.Emote) error {
	data, err := json.Marshal(emote)
	if err != nil {
		return err
	}
	return x.redis.RawClient().Set(ctx, cacheKeyEmotes+emote.ID.String(), string(data), 30*time.Second).Err()
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
