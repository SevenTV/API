package emoteset

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ResolverOps struct {
	types.Resolver
}

func NewOps(r types.Resolver) generated.EmoteSetOpsResolver {
	return &ResolverOps{r}
}

func (r *ResolverOps) Emotes(ctx context.Context, obj *model.EmoteSetOps, id primitive.ObjectID, action model.ListItemAction, nameArg *string) ([]*model.ActiveEmote, error) {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Get the emote
	emote, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": id}).First()
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return nil, errors.ErrUnknownEmote()
		}

		return nil, err
	}

	// Get the emote set
	name := ""
	if nameArg != nil {
		name = *nameArg
	}

	set, err := r.Ctx.Inst().Query.EmoteSets(ctx, bson.M{"_id": obj.ID}).First()
	if err != nil {
		if errors.Compare(err, errors.ErrNoItems()) {
			return nil, errors.ErrUnknownEmoteSet()
		}

		return nil, err
	}

	b := structures.NewEmoteSetBuilder(set)

	// Mutate the thing
	if err := r.Ctx.Inst().Mutate.EditEmotesInSet(ctx, b, mutate.EmoteSetMutationSetEmoteOptions{
		Actor: actor,
		Emotes: []mutate.EmoteSetMutationSetEmoteItem{{
			Action: structures.ListItemAction(action),
			ID:     id,
			Name:   name,
			Flags:  0,
		}},
	}); err != nil {
		return nil, err
	}

	// Clear cache keys for active sets / channel count
	k := r.Ctx.Inst().Redis.ComposeKey("gql-v3", fmt.Sprintf("emote:%s", id.Hex()))
	_, _ = r.Ctx.Inst().Redis.Del(ctx, k+":active_sets")
	_, _ = r.Ctx.Inst().Redis.Del(ctx, k+":channel_count")

	emoteIDs := make([]primitive.ObjectID, len(b.EmoteSet.Emotes))
	for i, e := range b.EmoteSet.Emotes {
		emoteIDs[i] = e.ID
	}

	// Publish an emote set update
	go func() {
		events.Publish(r.Ctx, "emote_sets", b.EmoteSet.ID)

		// Legacy Event API v1
		if set.Owner != nil && !actor.ID.IsZero() {
			tw, _, err := set.Owner.Connections.Twitch()
			if err != nil {
				return
			}

			if tw.EmoteSetID.IsZero() || tw.EmoteSetID != set.ID {
				return // skip if target emote set isn't bound to user connection
			}

			if err := events.PublishLegacyEventAPI(r.Ctx, action, tw.Data.Login, actor, set, emote); err != nil {
				zap.S().Errorw("redis",
					"error", err,
				)
			}
		}
	}()

	setModel := helpers.EmoteSetStructureToModel(b.EmoteSet, r.Ctx.Config().CdnURL)
	emotes, errs := r.Ctx.Inst().Loaders.EmoteByID().LoadAll(emoteIDs)

	for i, e := range emotes {
		if ae := setModel.Emotes[i]; ae != nil {
			setModel.Emotes[i].Emote = helpers.EmoteStructureToModel(e, r.Ctx.Config().CdnURL)
		}
	}

	return setModel.Emotes, multierror.Append(nil, errs...).ErrorOrNil()
}
