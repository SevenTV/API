package eventbridge

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	identifier_foreign_username = "foreign_username"
	identifier_foreign_id       = "foreign_id"
	identifier_username         = "username"
	identifier_id               = "id"
)

var userStateLoader *dataloader.DataLoader[string, structures.User]

func createUserStateLoader(gctx global.Context) {
	userStateLoader = dataloader.New(dataloader.Config[string, structures.User]{
		Fetch: func(keys []string) ([]structures.User, []error) {
			var (
				errs []error
				v    []structures.User
			)

			identifierMap := map[string]utils.Set[string]{
				identifier_foreign_username: {},
				identifier_foreign_id:       {},
				identifier_id:               {},
				identifier_username:         {},
			}

			for _, key := range keys {
				// Identify the target
				keysp := strings.SplitN(key, "|", 2)
				if len(keysp) != 2 {
					continue
				}

				platform := keysp[0]

				idsp := strings.SplitN(keysp[1], ":", 2)
				idType := idsp[0]
				identifier := idsp[1]

				// Platform specified: find by connection
				if platform != "" {
					switch idType {
					case "id":
						identifierMap["foreign_id"].Add(platform + ":" + identifier)
					case "username":
						identifierMap["foreign_username"].Add(platform + ":" + identifier)
					}
				} else { // no platform means app user
					switch idType {
					case "id":
						identifierMap["id"].Add(identifier)
					case "username":
						identifierMap["username"].Add(identifier)
					}
				}
			}

			wg := sync.WaitGroup{}
			mx := sync.Mutex{}

			for idType, identifiers := range identifierMap {
				if len(identifiers) == 0 {
					continue
				}

				wg.Add(1)

				go func(idType string, identifiers utils.Set[string]) {
					defer wg.Done()

					var (
						cur   *mongodb.Cursor
						err   error
						users = []structures.User{}
					)

					switch idType {
					case identifier_foreign_id, identifier_foreign_username:
						m := make(map[structures.UserConnectionPlatform][]string)

						for _, id := range identifiers.Values() {
							idsp := strings.SplitN(id, ":", 2)
							if len(idsp) != 2 {
								continue
							}

							platform := structures.UserConnectionPlatform((idsp[0]))
							id := idsp[1]

							m[platform] = append(m[platform], id)
						}

						for p, ids := range m {
							cur, err = gctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(gctx, bson.M{
								utils.Ternary(idType == identifier_foreign_id, "connections.id", "connections.data.login"): bson.M{
									"$in": ids,
								},
								"connections.platform": p,
							}, options.Find().SetProjection(bson.M{
								"_id":                                1,
								"avatar":                             1,
								"avatar_id":                          1,
								"username":                           1,
								"display_name":                       1,
								"connections.platform":               1,
								"connections.id":                     1,
								"connections.data.login":             1,
								"connections.data.profile_image_url": 1,
							}), options.Find().SetBatchSize(10))
						}
					case identifier_id:
						//iden := identifiers.Values()
						//idList := utils.Map(iden, func(x string) primitive.ObjectID {
						//	oid, err := primitive.ObjectIDFromHex(x)
						//	if err != nil {
						//		return primitive.NilObjectID
						//	}

						//	return oid
						//})

						// v, errs = gctx.Inst().Loaders.UserByID().LoadAll(idList)
					case identifier_username:
						// v, errs = gctx.Inst().Loaders.UserByUsername().LoadAll(identifiers.Values())
					}

					if cur == nil || err != nil {
						zap.S().Errorw("failed to load users for bridged cosmetics request command", "error", err)
						return
					}

					if err = cur.All(gctx, &users); err != nil {
						zap.S().Errorw("failed to load users for bridged cosmetics request command", "error", err)

						return
					}

					userMap := map[primitive.ObjectID]struct {
						i int
						u structures.User
					}{}
					userIDs := make([]primitive.ObjectID, len(users))

					for i, user := range users {
						userIDs[i] = user.ID

						userMap[user.ID] = struct {
							i int
							u structures.User
						}{
							i: i,
							u: user,
						}
					}

					entQuery := gctx.Inst().Query.Entitlements(gctx, bson.M{
						"user_id": bson.M{
							"$in": userIDs,
						},
					}, query.QueryEntitlementsOptions{
						SelectedOnly: true,
					})

					ents, err := entQuery.Items()
					if err != nil {
						zap.S().Errorw("failed to load entitlements for bridged cosmetics request command", "error", err)
					}

					roleMap := make(map[primitive.ObjectID]structures.Role)
					roles, err := gctx.Inst().Query.Roles(gctx, bson.M{})
					if err != nil {
						zap.S().Errorw("failed to load roles for bridged cosmetics request command", "error", err)
					}

					for _, role := range roles {
						roleMap[role.ID] = role
					}

					for _, ent := range ents {
						user := userMap[ent.UserID]

						for _, role := range ent.Roles {
							rol, ok := roleMap[role.Data.RefID]
							if !ok {
								continue
							}

							user.u.Roles = append(user.u.Roles, rol)
						}

						users[user.i] = user.u
					}

					mx.Lock()

					v = append(v, users...)

					mx.Unlock()

					for _, err := range errs {
						if err == nil || errors.Compare(err, errors.ErrUnknownUser()) {
							continue
						}

						zap.S().Errorw("failed to load users for bridged cosmetics request command", "error", err)

						break
					}
				}(idType, identifiers)
			}

			wg.Wait()

			return v, errs
		},
		Wait:     3000 * time.Millisecond,
		MaxBatch: 100,
	})
}

func handleUserState(gctx global.Context, ctx context.Context, body events.UserStateCommandBody) error {
	keys := make([]string, len(body.Identifiers))

	for i, id := range body.Identifiers {
		params := strings.Builder{}
		params.WriteString(string(body.Platform))
		params.WriteString("|")
		params.WriteString(id)

		keys[i] = params.String()
	}

	users, _ := userStateLoader.LoadAll(keys)

	var sid string
	switch t := ctx.Value(SESSION_ID_KEY).(type) {
	case string:
		sid = t
	}

	if sid == "" {
		zap.S().Errorw("failed to get session id from context")
		return nil
	}

	// Dispatch user avatar
	for _, user := range users {
		if (user.Avatar != nil || user.AvatarID != "") &&
			user.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation) {
			av := utils.ToJSON(gctx.Inst().Modelizer.Avatar(user))

			_ = gctx.Inst().Events.DispatchWithEffect(gctx, events.EventTypeCreateCosmetic, events.ChangeMap{
				ID:         user.ID,
				Kind:       structures.ObjectKindCosmetic,
				Contextual: true,
				Object:     av,
			}, events.DispatchOptions{
				Whisper: sid,
			})
		}
	}

	return nil
}
