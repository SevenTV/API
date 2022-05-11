package mutation

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (r *Resolver) AddChannelEmote(ctx context.Context, channelIDArg, emoteIDArg string, reasonArg *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Parse passed arguments
	channelID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	emoteID, er2 := primitive.ObjectIDFromHex(emoteIDArg)
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

	b := structures.NewEmoteSetBuilder(structures.EmoteSet{})
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{
		"_id": twConn.EmoteSetID,
	}).Decode(&b.EmoteSet); err != nil {
		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Run mutation
	if err = r.doSetChannelEmote(ctx, actor, emoteID, "", mutations.ListItemActionAdd, b); err != nil {
		graphql.AddError(ctx, err)
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) RemoveChannelEmote(ctx context.Context, channelIDArg, emoteIDArg string, reasonArg *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	// Parse passed arguments
	channelID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	emoteID, er2 := primitive.ObjectIDFromHex(emoteIDArg)
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
	b := structures.NewEmoteSetBuilder(structures.EmoteSet{})
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{
		"_id": twConn.EmoteSetID,
	}).Decode(&b.EmoteSet); err != nil {
		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Run mutation
	if err = r.doSetChannelEmote(ctx, actor, emoteID, "", mutations.ListItemActionRemove, b); err != nil {
		graphql.AddError(ctx, err)
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) EditChannelEmote(ctx context.Context, channelIDArg string, emoteIDArg string, data model.ChannelEmoteInput, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor == nil {
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
	b := structures.NewEmoteSetBuilder(structures.EmoteSet{})
	if err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{
		"_id": twConn.EmoteSetID,
	}).Decode(&b.EmoteSet); err != nil {
		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Run mutation
	if err = r.doSetChannelEmote(ctx, actor, emoteID, alias, mutations.ListItemActionUpdate, b); err != nil {
		graphql.AddError(ctx, err)
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) doSetChannelEmote(
	ctx context.Context,
	actor *structures.User,
	emoteID primitive.ObjectID,
	name string,
	action mutations.ListItemAction,
	b *structures.EmoteSetBuilder,
) error {
	if err := r.Ctx.Inst().Mutate.EditEmotesInSet(ctx, b, mutations.EmoteSetMutationSetEmoteOptions{
		Actor: actor,
		Emotes: []mutations.EmoteSetMutationSetEmoteItem{{
			Action: action,
			ID:     emoteID,
			Name:   name,
		}},
	}); err != nil {
		zap.S().Errorw("failed to update emotes in set",
			"error", err,
		)
		return err
	}

	// Publish an emote set update
	go func() {
		events.Publish(r.Ctx, "emote_sets", b.EmoteSet.ID)
	}()
	return nil
}
