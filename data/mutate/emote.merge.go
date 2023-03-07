package mutate

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
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
	if !actor.ID.IsZero() {
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

	if opt.NewEmote.ID.IsZero() {
		return errors.ErrInternalIncompleteMutation()
	}

	// Update all emote sets with the target emote active
	if _, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).UpdateMany(ctx, bson.M{
		"emotes.id": eb.Emote.ID,
		"$and": bson.A{bson.M{
			"emotes.id": bson.M{"$not": bson.M{"$eq": in.Emote.ID}},
		}},
	}, bson.M{"$set": bson.M{
		"emotes.$.id":             in.Emote.ID,
		"emotes.$.merged_from_id": eb.Emote.ID,
		"emotes.$.merged_at":      time.Now(),
	}}); err != nil {
		zap.S().Errorw("mongo, couldn't modify emote sets",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Delete the target emote
	if err := m.DeleteEmote(ctx, eb, DeleteEmoteOptions{
		Actor:          actor,
		VersionID:      opt.VersionID,
		ReplaceID:      in.Emote.ID,
		Reason:         opt.Reason,
		SkipValidation: false,
	}); err != nil {
		zap.S().Errorw("failed to delete the emote being merged",
			"error", err,
			"target_emote_id", eb.Emote.ID,
			"new_emote_id", in.Emote.ID,
		)
	}

	// Clear channel counts
	_, _ = m.redis.Del(ctx, m.redis.ComposeKey("gql-v3", fmt.Sprintf("emote:%s:active_sets", in.Emote.ID.Hex())))
	_, _ = m.redis.Del(ctx, m.redis.ComposeKey("gql-v3", fmt.Sprintf("emote:%s:channel_count", in.Emote.ID.Hex())))

	// Audit log
	c := structures.NewAuditChange("new_emote_id").WriteSingleValues("", in.Emote.ID.Hex())
	alb := structures.NewAuditLogBuilder(structures.AuditLog{
		Changes: []*structures.AuditLogChange{c},
		Reason:  opt.Reason,
	}).
		SetKind(structures.AuditLogKindMergeEmote).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmote).
		SetTargetID(eb.Emote.ID)

	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("mongo, couldn't insert audit log",
			"error", err,
		)
	}

	_, _ = m.cd.SendMessage("mod_actor_tracker", discordgo.MessageSend{
		Content: fmt.Sprintf("**[merge]** **[%s]** 🔀 [%s](%s) to [%s](%s) (reason: '%s')", actor.Username, eb.Emote.Name, eb.Emote.WebURL(m.id.Web), in.Emote.Name, in.Emote.WebURL(m.id.Web), opt.Reason),
	}, true)

	eb.MarkAsTainted()

	return nil
}

type MergeEmoteOptions struct {
	Actor    structures.User
	NewEmote structures.Emote
	// If specified, only this version will be merged
	//
	// by default, the latest version will be merged
	VersionID primitive.ObjectID
	// The reason given for the merger: will appear in audit logs
	Reason         string
	SkipValidation bool
}
