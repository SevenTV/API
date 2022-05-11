package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/internal/global"
	"go.mongodb.org/mongo-driver/bson"
)

func userLoader[T comparable](gCtx global.Context, keyName string) *dataloader.DataLoader[T, structures.User] {
	return dataloader.New(dataloader.Config[T, structures.User]{
		Wait: time.Millisecond * 25,
		Fetch: func(keys []T) ([]structures.User, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			models := make([]structures.User, len(keys))
			errs := make([]error, len(keys))

			// Fetch users
			users, _, err := gCtx.Inst().Query.SearchUsers(ctx, bson.M{
				keyName: bson.M{"$in": keys},
			})
			if err == nil {
				m := make(map[T]structures.User)
				for _, u := range users {
					switch keyName {
					case "username":
						m[utils.ToAny(u.Username).(T)] = u
					default:
						m[utils.ToAny(u.ID).(T)] = u
					}
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
