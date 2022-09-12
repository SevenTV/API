package mutation

import (
	"context"

	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/events"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/common/errors"
	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EditEmote(ctx context.Context, opt model.EmoteInput, reason *string) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Parse emote ID
	emoteID, err := primitive.ObjectIDFromHex(opt.ID)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Fetch the emote
	emotes, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": emoteID}).Items()
	if err != nil {
		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	if len(emotes) == 0 {
		return nil, errors.ErrUnknownEmote()
	}

	emote := emotes[0]
	version, _ := emote.GetVersion(emoteID)
	eb := structures.NewEmoteBuilder(emote)

	// Make edits
	if opt.Name != nil {
		eb.SetName(*opt.Name)
	}

	if opt.OwnerID != nil {
		ownerID, err := primitive.ObjectIDFromHex(*opt.OwnerID)
		if err != nil {
			return nil, errors.ErrBadObjectID()
		}

		eb.SetOwnerID(ownerID)
	}

	if opt.Visibility != nil {
		vis := int64(*opt.Visibility)
		flags := emote.Flags

		readModRequests := func() error {
			// Fetch mod request
			targetIDs := make([]primitive.ObjectID, len(emote.Versions))
			for i, ver := range emote.Versions {
				targetIDs[i] = ver.ID
			}

			result, err := r.Ctx.Inst().Query.ModRequestMessages(ctx, query.ModRequestMessagesQueryOptions{
				Actor: &actor,
				Targets: map[structures.ObjectKind]bool{
					structures.ObjectKindEmote: true,
				},
				Limit:     100,
				TargetIDs: targetIDs,
			}).Items()
			if err != nil && !errors.Compare(err, errors.ErrNoItems()) {
				return err
			}

			for _, msg := range result {
				mb := structures.NewMessageBuilder(msg)
				// Mark the message as read
				_, err := r.Ctx.Inst().Mutate.SetMessageReadStates(ctx, mb, true, mutate.MessageReadStateOptions{
					Actor: &actor,
				})
				if err != nil {
					return err
				}
			}

			return nil
		}

		// listed
		if !version.State.Listed && !utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityUnlisted)) {
			if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
				return nil, errors.ErrInsufficientPrivilege().SetDetail("Not allowed to list this emote")
			}

			version.State.Listed = true
			eb.UpdateVersion(version.ID, version)

			// Handle legacy moderation
			// This was how emotes were approved in v2,
			// so we must clear the Mod Request.
			if err = readModRequests(); err != nil {
				return nil, err
			}
		} else if !version.State.Listed && utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityPermanentlyUnlisted)) {
			// Handle legacy moderation
			// "Permanently unlisted" flag means reading the
			// mod request without listing the emote
			if err = readModRequests(); err != nil {
				return nil, err
			}
		} else if version.State.Listed && utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityUnlisted)) {
			if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
				return nil, errors.ErrInsufficientPrivilege().SetDetail("Not allowed to unlist this emote")
			}
			version.State.Listed = false
			eb.UpdateVersion(version.ID, version)
		}

		// zero-width
		if emote.HasFlag(structures.EmoteFlagsZeroWidth) && !utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityZeroWidth)) {
			flags &= ^structures.EmoteFlagsZeroWidth
		} else if !emote.HasFlag(structures.EmoteFlagsZeroWidth) && utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityZeroWidth)) {
			flags |= structures.EmoteFlagsZeroWidth
		}
		// privacy
		if emote.HasFlag(structures.EmoteFlagsPrivate) && !utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityPrivate)) {
			flags &= ^structures.EmoteFlagsPrivate
		} else if !emote.HasFlag(structures.EmoteFlagsPrivate) && utils.BitField.HasBits(vis, int64(v2structures.EmoteVisibilityPrivate)) {
			flags |= structures.EmoteFlagsPrivate
		}

		eb.SetFlags(flags)
	}

	if err = r.Ctx.Inst().Mutate.EditEmote(ctx, eb, mutate.EmoteEditOptions{
		Actor: &actor,
	}); err != nil {
		return nil, err
	}

	go func() {
		events.Publish(r.Ctx, "emotes", emoteID)
	}()

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) DeleteEmote(ctx context.Context, id string, reason string) (*bool, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Parse emote ID
	emoteID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Fetch the emote
	emote, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": emoteID}).First()
	if err != nil {
		return nil, errors.ErrUnknownEmote()
	}

	eb := structures.NewEmoteBuilder(emote)

	if err = r.Ctx.Inst().Mutate.DeleteEmote(ctx, eb, mutate.DeleteEmoteOptions{
		Actor:  &actor,
		Reason: reason,
	}); err != nil {
		return nil, err
	}

	go func() {
		events.Publish(r.Ctx, "emotes", emoteID)
	}()

	return utils.PointerOf(true), nil
}

// MergeEmote implements generated.MutationResolver
func (r *Resolver) MergeEmote(ctx context.Context, oldIDArg string, newIDArg string, reason string) (*model.Emote, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Parse emote ID
	oldID, err := primitive.ObjectIDFromHex(oldIDArg)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	newID, err := primitive.ObjectIDFromHex(newIDArg)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	// Fetch the emote
	emotes, err := r.Ctx.Inst().Query.Emotes(ctx, bson.M{"versions.id": bson.M{"$in": bson.A{oldID, newID}}}).Items()
	if err != nil {
		return nil, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	if len(emotes) != 2 {
		return nil, errors.ErrUnknownEmote()
	}

	var oldEmote structures.Emote

	var newEmote structures.Emote

	for _, e := range emotes {
		switch e.ID {
		case oldID:
			oldEmote = e
		case newID:
			newEmote = e
		}
	}

	if err := r.Ctx.Inst().Mutate.MergeEmote(ctx, structures.NewEmoteBuilder(oldEmote), mutate.MergeEmoteOptions{
		Actor:          &actor,
		NewEmote:       newEmote,
		Reason:         reason,
		SkipValidation: false,
	}); err != nil {
		return nil, err
	}

	return helpers.EmoteStructureToModel(newEmote, r.Ctx.Config().CdnURL), nil
}
