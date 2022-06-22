package user

import (
	"context"
	"time"

	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/mutations"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.UserOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Connections(ctx context.Context, obj *model.UserOps, id string, d model.UserConnectionUpdate) ([]*model.UserConnection, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	b := structures.NewUserBuilder(structures.DeletedUser)
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
		"_id": obj.ID,
	}).Decode(&b.User); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnknownUser()
		}

		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Perform a mutation
	var err error
	if d.EmoteSetID != nil {
		conn, _, err := b.User.Connections.Twitch()
		if err != nil {
			return nil, errors.ErrUnknownUserConnection()
		}

		oldSetID := conn.EmoteSetID

		setID := *d.EmoteSetID

		if err = r.Ctx.Inst().Mutate.SetUserConnectionActiveEmoteSet(ctx, b, mutations.SetUserActiveEmoteSet{
			EmoteSetID:   *d.EmoteSetID,
			Platform:     structures.UserConnectionPlatformTwitch,
			Actor:        actor,
			ConnectionID: id,
		}); err != nil {
			zap.S().Errorw("failed to update user's active emote set",
				"error", err,
				"connection_id", conn.ID,
			)

			return nil, err
		}

		// Send legacy events
		if conn.Platform == structures.UserConnectionPlatformTwitch {
			sets := r.Ctx.Inst().Query.EmoteSets(ctx, bson.M{"_id": bson.M{"$in": bson.A{setID, oldSetID}}})
			if !sets.Empty() {
				go func() {
					var (
						newSet structures.EmoteSet
						oldSet structures.EmoteSet
					)

					items, _ := sets.Items()
					for _, es := range items {
						switch es.ID {
						case setID:
							newSet = es
						case oldSetID:
							oldSet = es
						}
					}

					if !newSet.ID.IsZero() {
						if !oldSet.ID.IsZero() {
							// Send "REMOVE" events to former set
							for _, ae := range oldSet.Emotes {
								if ae.Emote == nil {
									continue
								}

								if err := events.PublishLegacyEventAPI(r.Ctx, model.ListItemActionRemove, conn.Data.Login, *actor, oldSet, *ae.Emote); err != nil {
									zap.S().Errorw("redis",
										"error", err,
									)
								}

								time.Sleep(time.Millisecond * 10) // todo
							}
						}

						for _, ae := range newSet.Emotes {
							if ae.Emote == nil {
								continue
							}

							if err := events.PublishLegacyEventAPI(r.Ctx, model.ListItemActionAdd, conn.Data.Login, *actor, oldSet, *ae.Emote); err != nil {
								zap.S().Errorw("redis",
									"error", err,
								)
							}

							time.Sleep(time.Millisecond * 10) // todo
						}
					}
				}()
			}
		}
	}

	if err != nil {
		return nil, err
	}

	result := helpers.UserStructureToModel(b.User, r.Ctx.Config().CdnURL)
	events.Publish(r.Ctx, "users", b.User.ID)

	return result.Connections, nil
}
