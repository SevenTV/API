package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/dataloader"
	"github.com/SevenTV/Common/errors"
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

			items := make([]structures.User, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted emotes in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedUser
			}

			// Fetch users
			result := gCtx.Inst().Query.Users(ctx, bson.M{
				keyName: bson.M{"$in": keys},
			})
			if result.Empty() {
				return items, errs
			}
			users, err := result.Items()

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
						items[i] = x
					} else {
						errs[i] = errors.ErrUnknownUser()
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
