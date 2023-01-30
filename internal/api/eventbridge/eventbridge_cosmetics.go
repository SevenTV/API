package eventbridge

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	identifier_foreign_username = "foreign_username"
	identifier_foreign_id       = "foreign_id"
	identifier_username         = "username"
	identifier_id               = "id"
)

func handleCosmetics(gctx global.Context, ctx context.Context, body events.CosmeticsCommandBody) error {
	var sid string
	switch v := ctx.Value(SESSION_ID_KEY).(type) {
	case string:
		sid = v
	}

	identifierMap := map[string]utils.Set[string]{
		identifier_foreign_username: {},
		identifier_foreign_id:       {},
		identifier_id:               {},
		identifier_username:         {},
	}

	for _, id := range body.Identifiers {
		// Identify the target
		idsp := strings.SplitN(id, ":", 2)
		if len(idsp) != 2 {
			return errors.ErrInvalidRequest().SetDetail("Invalid Identifier Format")
		}

		idType := idsp[0]
		identifier := idsp[1]

		// Platform specified: find by connection
		if body.Platform != "" {
			switch idType {
			case "id":
				identifierMap["foreign_id"].Add(identifier)
			case "username":
				identifierMap["foreign_username"].Add(identifier)
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

	users := []structures.User{}

	for idType, identifiers := range identifierMap {
		if len(identifiers) == 0 {
			continue
		}

		wg.Add(1)

		go func(idType string, identifiers utils.Set[string]) {
			var (
				errs []error
				v    []structures.User
			)

			switch idType {
			case identifier_foreign_id:
				v, errs = gctx.Inst().Loaders.UserByConnectionID(body.Platform).LoadAll(identifiers.Values())
			case identifier_foreign_username:
				v, errs = gctx.Inst().Loaders.UserByConnectionUsername(body.Platform).LoadAll(identifiers.Values())
			case identifier_id:
				iden := identifiers.Values()
				idList := utils.Map(iden, func(x string) primitive.ObjectID {
					oid, err := primitive.ObjectIDFromHex(x)
					if err != nil {
						return primitive.NilObjectID
					}

					return oid
				})

				v, errs = gctx.Inst().Loaders.UserByID().LoadAll(idList)
			case identifier_username:
				v, errs = gctx.Inst().Loaders.UserByUsername().LoadAll(identifiers.Values())
			}

			mx.Lock()
			users = append(users, v...)
			mx.Unlock()

			wg.Done()

			if len(errs) > 0 {
				zap.S().Errorw("failed to load users for bridged cosmetics request command", "errors", multierror.Append(nil, errs...).Error())
			}
		}(idType, identifiers)
	}

	wg.Wait()

	kinds := utils.Set[structures.CosmeticKind]{}
	kinds.Fill(body.Kinds...)

	// TODO: create a utility to dispatch with a redis pipeline

	// Dispatch user avatar
	for _, user := range users {
		if kinds.Has(structures.CosmeticKindAvatar) {
			av := gctx.Inst().Modelizer.Avatar(user)

			if _, err := gctx.Inst().Events.DispatchWithEffect(ctx, events.EventTypeCreateCosmetic, events.ChangeMap{
				ID:         user.ID,
				Kind:       structures.ObjectKindCosmetic,
				Contextual: true,
				Object:     utils.ToJSON(av),
			}, events.DispatchOptions{
				Whisper: sid,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
