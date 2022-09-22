package mutation

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	model3 "github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (r *Resolver) AddChannelEmote(ctx context.Context, channelIDArg, emoteIDArg string, reasonArg *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Parse passed arguments
	channelID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	emoteID, er2 := primitive.ObjectIDFromHex(emoteIDArg)

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(emoteID)
	if err != nil {
		return nil, err
	}

	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Get the target user
	target, err := r.Ctx.Inst().Loaders.UserByID().Load(channelID)
	if err != nil {
		return nil, err
	}

	// Get the emote set
	twConn, _, err := target.Connections.Twitch()
	if err != nil {
		return nil, errors.ErrUnknownEmoteSet()
	}

	es, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(twConn.EmoteSetID)
	if err != nil {
		return nil, err
	}

	if es.ID.IsZero() {
		esb := structures.NewEmoteSetBuilder(es)
		esb.EmoteSet.ID = primitive.NewObjectIDFromTimestamp(time.Now())
		esb.EmoteSet.Emotes = []structures.ActiveEmote{}
		esb.SetOwnerID(target.ID).
			SetName(fmt.Sprintf("%s's Emotes", target.DisplayName)).
			SetCapacity(250)

		if err = r.Ctx.Inst().Mutate.CreateEmoteSet(ctx, esb, mutate.EmoteSetMutationOptions{
			Actor: actor,
		}); err != nil {
			return nil, err
		}

		// Assign the new set to each of the user's connections
		for _, conn := range target.Connections {
			if conn.EmoteSlots == 0 {
				continue // skip if connection doesn't support emotes
			}

			oldSet, _ := r.Ctx.Inst().Loaders.EmoteSetByID().Load(conn.EmoteSetID)

			ub := structures.NewUserBuilder(target)
			if err = r.Ctx.Inst().Mutate.SetUserConnectionActiveEmoteSet(ctx, ub, mutate.SetUserActiveEmoteSet{
				NewSet:       esb.EmoteSet,
				OldSet:       oldSet,
				Platform:     structures.UserConnectionPlatformTwitch,
				Actor:        actor,
				ConnectionID: conn.ID,
			}); err != nil {
				return nil, err
			}
		}

		es = esb.EmoteSet
	}

	esb := structures.NewEmoteSetBuilder(es)

	// Run mutation
	if err = r.doSetChannelEmote(ctx, &actor, emoteID, "", structures.ListItemActionAdd, esb); err != nil {
		graphql.AddError(ctx, err)
		return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
	}

	go func() {
		if err := events.PublishLegacyEventAPI(r.Ctx, model3.ListItemActionAdd, twConn.Data.Login, actor, esb.EmoteSet, emote); err != nil {
			zap.S().Errorw("redis",
				"error", err,
			)
		}
	}()

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) RemoveChannelEmote(ctx context.Context, channelIDArg, emoteIDArg string, reasonArg *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Parse passed arguments
	channelID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	emoteID, er2 := primitive.ObjectIDFromHex(emoteIDArg)

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(emoteID)
	if err != nil {
		return nil, err
	}

	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Get the target user
	target, err := r.Ctx.Inst().Loaders.UserByID().Load(channelID)
	if err != nil {
		return nil, err
	}

	// Get the emote set
	twConn, _, err := target.Connections.Twitch()
	if err != nil {
		return nil, errors.ErrUnknownEmoteSet()
	}

	es, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(twConn.EmoteSetID)
	if err != nil {
		return nil, err
	}

	esb := structures.NewEmoteSetBuilder(es)

	// Run mutation
	if err = r.doSetChannelEmote(ctx, &actor, emoteID, "", structures.ListItemActionRemove, esb); err != nil {
		graphql.AddError(ctx, err)
		return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
	}

	go func() {
		if err := events.PublishLegacyEventAPI(r.Ctx, model3.ListItemActionRemove, twConn.Data.Login, actor, esb.EmoteSet, emote); err != nil {
			zap.S().Errorw("redis",
				"error", err,
			)
		}
	}()

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) EditChannelEmote(ctx context.Context, channelIDArg string, emoteIDArg string, data model.ChannelEmoteInput, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Data must contain alias
	alias := ""
	if data.Alias != nil {
		alias = *data.Alias
	}

	// Parse passed arguments
	channelID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	emoteID, er2 := primitive.ObjectIDFromHex(emoteIDArg)

	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(emoteID)
	if err != nil {
		return nil, err
	}

	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Get the target user
	target, err := r.Ctx.Inst().Loaders.UserByID().Load(channelID)
	if err != nil {
		return nil, err
	}

	// Get the emote set
	twConn, _, err := target.Connections.Twitch()
	if err != nil {
		return nil, errors.ErrUnknownEmoteSet()
	}

	es, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(twConn.EmoteSetID)
	if err != nil {
		return nil, err
	}

	esb := structures.NewEmoteSetBuilder(es)

	// Run mutation
	if err = r.doSetChannelEmote(ctx, &actor, emoteID, alias, structures.ListItemActionUpdate, esb); err != nil {
		graphql.AddError(ctx, err)
		return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
	}

	go func() {
		if err := events.PublishLegacyEventAPI(r.Ctx, model3.ListItemActionUpdate, twConn.Data.Login, actor, esb.EmoteSet, emote); err != nil {
			zap.S().Errorw("redis",
				"error", err,
			)
		}
	}()

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) doSetChannelEmote(
	ctx context.Context,
	actor *structures.User,
	emoteID primitive.ObjectID,
	name string,
	action structures.ListItemAction,
	b *structures.EmoteSetBuilder,
) error {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	if actor == nil {
		return errors.ErrUnauthorized()
	}

	if err := r.Ctx.Inst().Mutate.EditEmotesInSet(ctx, b, mutate.EmoteSetMutationSetEmoteOptions{
		Actor: *actor,
		Emotes: []mutate.EmoteSetMutationSetEmoteItem{{
			Action: action,
			ID:     emoteID,
			Name:   name,
		}},
	}); err != nil {
		return err
	}

	// Publish an emote set update
	go func() {
		events.Publish(r.Ctx, "emote_sets", b.EmoteSet.ID)
	}()

	return nil
}
