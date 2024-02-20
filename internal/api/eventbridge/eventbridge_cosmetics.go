package eventbridge

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/dataloader"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

const (
	identifier_foreign_username = "foreign_username"
	identifier_foreign_id       = "foreign_id"
	identifier_username         = "username"
	identifier_id               = "id"
)

type UserIdentifier struct {
	Platform structures.UserConnectionPlatform `json:"platform"`
	IdType   string                            `json:"id_type"`
	Id       string                            `json:"id"`
}

var userStateLoader *dataloader.DataLoader[UserIdentifier, structures.User]

type identifier struct {
	Platform structures.UserConnectionPlatform
	Id       string
}

func createUserStateLoader(gctx global.Context) {
	userStateLoader = dataloader.New(dataloader.Config[UserIdentifier, structures.User]{
		Fetch: func(keys []UserIdentifier) ([]structures.User, []error) {
			var (
				errs      []error
				resultMap map[UserIdentifier]structures.User = map[UserIdentifier]structures.User{}
			)

			identifierMap := map[string]utils.Set[identifier]{
				identifier_foreign_username: {},
				identifier_foreign_id:       {},
			}

			for _, key := range keys {
				// Platform specified: find by connection
				if key.Platform != "" {
					switch key.IdType {
					case identifier_id:
						identifierMap[identifier_foreign_id].Add(identifier{Platform: key.Platform, Id: key.Id})
					case identifier_username:
						identifierMap[identifier_foreign_username].Add(identifier{Platform: key.Platform, Id: key.Id})
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

				go func(idType string, identifiers utils.Set[identifier]) {
					defer wg.Done()

					var users = []structures.User{}

					switch idType {
					case identifier_foreign_id, identifier_foreign_username:
						l := utils.Ternary(idType == identifier_foreign_id, gctx.Inst().Loaders.UserByConnectionID, gctx.Inst().Loaders.UserByConnectionUsername)

						m := map[structures.UserConnectionPlatform][]string{}

						for _, id := range identifiers.Values() {
							m[id.Platform] = append(m[id.Platform], id.Id)
						}

						for p, ids := range m {
							users, errs = l(p).LoadAll(ids)
							mx.Lock()

							for i := range users {
								if errs[i] != nil {
									if errors.Compare(errs[i], errors.ErrUnknownUser()) {
										continue
									}

									zap.S().Errorw("failed to load user for bridged cosmetics request command", "error", errs[i])
									break
								}

								id := ids[i]
								key := UserIdentifier{
									Platform: p,
									IdType:   utils.Ternary(idType == identifier_foreign_id, identifier_id, identifier_username),
									Id:       id,
								}

								resultMap[key] = users[i]
							}

							mx.Unlock()
						}
					}
				}(idType, identifiers)
			}

			wg.Wait()

			result := make([]structures.User, len(keys))
			for i, key := range keys {
				result[i] = resultMap[key]
			}

			return result, errs
		},
		Wait:     250 * time.Millisecond,
		MaxBatch: 500,
	})
}

func handleUserState(gctx global.Context, ctx context.Context, body events.BridgedCommandBody) ([]events.Message[json.RawMessage], error) {
	keys := make([]UserIdentifier, len(body.Identifiers))

	for i, id := range body.Identifiers {
		splits := strings.SplitN(id, ":", 2)

		if len(splits) != 2 {
			zap.S().Errorw("invalid user identifier", "identifier", id)
			return nil, nil
		}

		keys[i] = UserIdentifier{
			Platform: body.Platform,
			IdType:   splits[0],
			Id:       splits[1],
		}
	}

	users, _ := userStateLoader.LoadAll(keys)
	result := []events.Message[json.RawMessage]{}

	// Dispatch user avatar
	for _, user := range users {
		if (user.Avatar != nil || user.AvatarID != "") &&
			user.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation) {
			av := utils.ToJSON(gctx.Inst().Modelizer.Avatar(user))

			result = append(result, events.NewMessage(events.OpcodeDispatch, events.DispatchPayload{
				Type: events.EventTypeCreateCosmetic,
				Body: events.ChangeMap{
					ID:         user.ID,
					Kind:       structures.ObjectKindCosmetic,
					Contextual: true,
					Object:     av,
				},
			}).ToRaw())
		}
	}

	return result, nil
}
