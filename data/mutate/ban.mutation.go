package mutate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (m *Mutate) CreateBan(ctx context.Context, bb *structures.BanBuilder, opt CreateBanOptions) error {
	if bb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if bb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	if opt.Victim == nil {
		return errors.ErrMissingRequiredField().SetDetail("Did not specify a victim")
	}

	// Check permissions
	// can the actor ban the victim?
	actorID := primitive.NilObjectID
	actor := opt.Actor
	victim := opt.Victim

	if !opt.SkipValidation {
		actorID = actor.ID

		if victim.ID == actor.ID {
			return errors.ErrDontBeSilly()
		}

		if !actor.HasPermission(structures.RolePermissionManageBans) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
				"MISSING_PERMISSION": "MANAGE_BANS",
			})
		}

		if victim.GetHighestRole().Position >= actor.GetHighestRole().Position {
			return errors.ErrInsufficientPrivilege().
				SetDetail("Victim has an equal or higher privilege level").
				SetFields(errors.Fields{
					"ACTOR_ROLE_POSITION":  actor.GetHighestRole().Position,
					"VICTIM_ROLE_POSITION": victim.GetHighestRole().Position,
				})
		}
	}

	// Write
	result, err := m.mongo.Collection(mongo.CollectionNameBans).InsertOne(ctx, bb.Ban)
	if err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	switch t := result.InsertedID.(type) {
	case primitive.ObjectID:
		bb.Ban.ID = t
	}

	// Get the newly created ban
	_ = m.mongo.Collection(mongo.CollectionNameBans).FindOne(ctx, bson.M{"_id": bb.Ban.ID}).Decode(bb.Ban)

	// Send a message to the victim
	mb := structures.NewMessageBuilder(structures.Message[structures.MessageDataInbox]{}).
		SetKind(structures.MessageKindInbox).
		SetAuthorID(actorID).
		SetTimestamp(time.Now()).
		SetAnonymous(opt.AnonymousActor).
		SetData(structures.MessageDataInbox{
			Subject:   "inbox.generic.client_banned.subject",
			Content:   "inbox.generic.client_banned.content",
			Important: true,
			Placeholders: func() map[string]string {
				m := map[string]string{
					"BAN_REASON":    bb.Ban.Reason,
					"BAN_EXPIRE_AT": utils.Ternary(bb.Ban.ExpireAt.IsZero(), "never", bb.Ban.ExpireAt.Format(time.RFC822)),
				}
				for k, e := range structures.BanEffectMap {
					if bb.Ban.Effects.Has(e) {
						m[fmt.Sprintf("EFFECT_%s", k)] = fmt.Sprintf(
							"inbox.generic.client_banned.effect.%s", strings.ToLower(k),
						)
					}
				}
				return m
			}(),
		})
	if err := m.SendInboxMessage(ctx, mb, SendInboxMessageOptions{
		Actor:                actor,
		Recipients:           []primitive.ObjectID{victim.ID},
		ConsiderBlockedUsers: false,
	}); err != nil {
		zap.S().Errorw("failed to send inbox message to victim about created ban",
			"error", err,
			"actor_id", actor.ID.Hex(),
			"victim_id", victim.ID.Hex(),
			"ban_id", bb.Ban.ID.Hex(),
		)
	}

	_, _ = m.cd.SendMessage("mod_actor_tracker", discordgo.MessageSend{
		Content: fmt.Sprintf("**[ban]** **[%s]** 🔨 [%s](%s) until %s for reason '%s', with these effects: %s",
			actor.Username, victim.Username,
			victim.WebURL(m.id.Web),
			bb.Ban.ExpireAt.Format(time.RFC1123),
			bb.Ban.Reason,
			bb.Ban.Effects.String(),
		),
	}, true)

	bb.MarkAsTainted()

	return nil
}

type CreateBanOptions struct {
	Actor          *structures.User
	AnonymousActor bool
	Victim         *structures.User
	SkipValidation bool
}

func (m *Mutate) EditBan(ctx context.Context, bb *structures.BanBuilder, opt EditBanOptions) error {
	if bb == nil || bb.Ban.ID.IsZero() {
		return errors.ErrInternalIncompleteMutation()
	} else if bb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	actor := opt.Actor
	if !opt.SkipValidation {
		if !actor.HasPermission(structures.RolePermissionManageBans) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
				"MISSING_PERMISSION": "MANAGE_BANS",
			})
		}
	}

	// Write the change
	if _, err := m.mongo.Collection(mongo.CollectionNameBans).UpdateOne(ctx, bson.M{"_id": bb.Ban.ID}, bb.Update); err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	return nil
}

type EditBanOptions struct {
	Actor          *structures.User
	SkipValidation bool
}
