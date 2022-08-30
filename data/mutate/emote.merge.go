package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (m *Mutate) MergeEmote(ctx context.Context, eb *structures.EmoteBuilder, opt MergeEmoteOptions) error {
	if eb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if eb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	in := structures.NewEmoteBuilder(opt.NewEmote)

	// Check actor permissions
	actor := opt.Actor
	// Check actor's permission
	if actor != nil {
		// User is not privileged
		if !actor.HasPermission(structures.RolePermissionEditAnyEmote) {
			if eb.Emote.OwnerID.IsZero() { // Deny when emote has no owner
				return errors.ErrInsufficientPrivilege()
			}

			// Check if actor is editor of the emote owner
			isPermittedEditor := false

			for _, ed := range actor.EditorOf {
				if ed.ID != eb.Emote.OwnerID {
					continue
				}
				// Allow if the actor has the "manage owned emotes" permission
				// as the editor of the emote owner
				if ed.HasPermission(structures.UserEditorPermissionManageOwnedEmotes) {
					isPermittedEditor = true
					break
				}
			}

			if eb.Emote.OwnerID != actor.ID && !isPermittedEditor { // Deny when not the owner or editor of the owner of the emote
				return errors.ErrInsufficientPrivilege()
			}
		}
	} else if !opt.SkipValidation {
		// if validation is not skipped then an Actor is mandatory
		return errors.ErrUnauthorized()
	}

	// Is this a silly request?
	if eb.Emote.ID == in.Emote.ID {
		return errors.ErrDontBeSilly().SetDetail("It's not possible to merge an emote into itself")
	}

	// Update all emote sets with the target emote active
	if _, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).UpdateMany(ctx, bson.M{
		"emotes.id": eb.Emote.ID,
		"$and": bson.A{bson.M{
			"emotes.id": bson.M{"$not": bson.M{"$eq": in.Emote.ID}},
		}},
	}, bson.M{"$set": bson.M{
		"emotes.$.id": in.Emote.ID,
	}}); err != nil {
		zap.S().Errorw("mongo, couldn't modify emote sets",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// TODO: Delete the target emote
	if err := m.DeleteEmote(ctx, eb, DeleteEmoteOptions{
		Actor:          actor,
		VersionID:      opt.VersionID,
		Reason:         "",
		SkipValidation: false,
	}); err != nil {
		zap.S().Errorw("failed to delete the emote being merged",
			"error", err,
			"target_emote_id", eb.Emote.ID,
			"new_emote_id", in.Emote.ID,
		)
	}

	eb.MarkAsTainted()

	return nil
}

type MergeEmoteOptions struct {
	Actor    *structures.User
	NewEmote structures.Emote
	// If specified, only this version will be merged
	//
	// by default, the latest version will be merged
	VersionID primitive.ObjectID
	// The reason given for the merger: will appear in audit logs
	Reason         string
	SkipValidation bool
}
