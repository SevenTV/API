package loaders

import (
	"context"
	"sync"
	"time"

	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func emoteLoader(ctx context.Context, x inst, key string) EmoteLoaderByID {
	go initCache()

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
			cachedEmotes := getEmotesFromCache(keys)

			remainingKeys := []primitive.ObjectID{}

			for i, key := range keys {
				if emote, ok := cachedEmotes[key]; ok {
					items[i] = emote
					continue
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

						// store emote in cache
						setEmoteInCache(emote)
					}
				}
			}

			return items, errs
		},
	})
}

// TODO: clean up the code for cache to make it more universal for other loaders if needed
var emoteCache = make(map[primitive.ObjectID]cachedEmote)
var cacheMx = &sync.Mutex{}

type cachedEmote struct {
	emote  structures.Emote
	expire time.Time
}

func initCache() {
	for range time.Tick(time.Minute) {
		cacheMx.Lock()
		for k, v := range emoteCache {
			if time.Now().After(v.expire) {
				delete(emoteCache, k)
			}
		}
		cacheMx.Unlock()
	}
}

func getEmotesFromCache(keys []primitive.ObjectID) map[primitive.ObjectID]structures.Emote {
	cacheMx.Lock()
	defer cacheMx.Unlock()

	emotes := make(map[primitive.ObjectID]structures.Emote)

	for _, key := range keys {
		emote, ok := emoteCache[key]
		if !ok {
			continue
		}

		emotes[key] = emote.emote
	}

	return emotes
}

func setEmoteInCache(emote structures.Emote) {
	cacheMx.Lock()
	defer cacheMx.Unlock()

	emoteCache[emote.ID] = cachedEmote{
		emote:  emote,
		expire: time.Now().Add(time.Minute * 5),
	}
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
