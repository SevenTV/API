package loaders

import (
	"context"
	"fmt"
	"time"

	"github.com/seventv/api/data/query"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func userLoader[T comparable](ctx context.Context, x inst, keyName string) *dataloader.DataLoader[T, structures.User] {
	return dataloader.New(dataloader.Config[T, structures.User]{
		Wait:     time.Millisecond * 25,
		MaxBatch: 250,
		Fetch: func(keys []T) ([]structures.User, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			items := make([]structures.User, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted users in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedUser
			}

			// Fetch users
			result := x.query.Users(ctx, bson.M{
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
						v, _ := utils.ToAny(u.Username).(T)
						m[v] = u
					default:
						v, _ := utils.ToAny(u.ID).(T)
						m[v] = u
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

func userByConnectionLoader(ctx context.Context, x inst, platform structures.UserConnectionPlatform, key string) *dataloader.DataLoader[string, structures.User] {
	return dataloader.New(dataloader.Config[string, structures.User]{
		Wait:     time.Millisecond * 25,
		MaxBatch: 500,
		Fetch: func(keys []string) ([]structures.User, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			items := make([]structures.User, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted users in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedUser
			}

			// Fetch users
			result := x.query.Users(ctx, bson.M{
				fmt.Sprintf("connections.%s", key): bson.M{"$in": keys},
				"connections.platform":             platform,
			})
			if result.Empty() {
				return items, errs
			}
			users, err := result.Items()

			if err == nil {
				m := make(map[string]structures.User)
				for _, u := range users {
					switch key {
					case "data.login", "data.username":
						for _, c := range u.Connections {
							username, _ := c.Username()

							if c.Platform == platform {
								m[username] = u
							}
						}
					default:
						for _, c := range u.Connections {
							if c.Platform == platform {
								m[c.ID] = u
							}
						}
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

func entitlementsLoader(ctx context.Context, x inst) *dataloader.DataLoader[primitive.ObjectID, query.EntitlementQueryResult] {
	return dataloader.New(dataloader.Config[primitive.ObjectID, query.EntitlementQueryResult]{
		Wait: time.Millisecond * 100,
		Fetch: func(keys []primitive.ObjectID) ([]query.EntitlementQueryResult, []error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			items := make([]query.EntitlementQueryResult, len(keys))
			errs := make([]error, len(keys))

			// Fetch entitlements
			result := x.query.Entitlements(ctx, bson.M{
				"user_id": bson.M{"$in": keys},
			}, query.QueryEntitlementsOptions{SelectedOnly: true})
			if result.Empty() {
				return items, errs
			}
			entitlements, err := result.Items()

			if err == nil {
				m := make(map[primitive.ObjectID]query.EntitlementQueryResult)

				for _, e := range entitlements {
					m[e.UserID] = e
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
