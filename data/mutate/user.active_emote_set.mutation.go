package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *Mutate) SetUserConnectionActiveEmoteSet(ctx context.Context, ub *structures.UserBuilder, opt SetUserActiveEmoteSet) error {
	if ub == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if ub.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	// Check for actor's permission to do this
	actor := opt.Actor
	victim := &ub.User

	if !opt.SkipValidation {
		if actor.ID != victim.ID { // actor is modfiying another user
			notPrivileged := errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to change the active Emote Set of this user")

			if !actor.HasPermission(structures.RolePermissionManageUsers) { // actor is not a moderator
				ed, ok, _ := victim.GetEditor(actor.ID)
				if !ok { // actor is not an editor of the victim
					return notPrivileged
				}

				if !ed.HasPermission(structures.UserEditorPermissionManageEmoteSets) { // actor lacks the necessary permission
					return notPrivileged
				}
			}
		}

		// Validate that the emote set exists and can be enabled
		if !opt.EmoteSetID.IsZero() {
			set := &structures.EmoteSet{}
			if err := m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{
				"_id": opt.EmoteSetID,
			}).Decode(set); err != nil {
				if err == mongo.ErrNoDocuments {
					return errors.ErrUnknownEmoteSet()
				}

				return errors.ErrInternalServerError().SetDetail(err.Error())
			}

			if !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) && set.OwnerID != victim.ID {
				return errors.ErrInsufficientPrivilege().
					SetFields(errors.Fields{"owner_id": set.OwnerID.Hex()}).
					SetDetail("You cannot assign another user's Emote Set to your channel")
			}
		}
	}

	// Get the connection
	conn := ub.GetConnection(opt.Platform, opt.ConnectionID)
	if conn == nil {
		return errors.ErrUnknownUserConnection()
	}

	conn.SetActiveEmoteSet(opt.EmoteSetID)

	// Update document
	if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(
		ctx,
		bson.M{
			"_id":            victim.ID,
			"connections.id": opt.ConnectionID,
		},
		conn.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(victim); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrUnknownUser().SetDetail("Victim was not found and could not be updated")
		}

		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	ub.MarkAsTainted()

	return nil
}

type SetUserActiveEmoteSet struct {
	EmoteSetID     primitive.ObjectID
	Platform       structures.UserConnectionPlatform
	Actor          *structures.User
	ConnectionID   string
	SkipValidation bool
}
