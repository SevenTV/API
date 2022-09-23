package mutate

import (
	"context"
	"time"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const MAXIMUM_ALLOWED_EMOTE_SETS = 10

// Create: create the new emote set
func (m *Mutate) CreateEmoteSet(ctx context.Context, esb *structures.EmoteSetBuilder, opt EmoteSetMutationOptions) error {
	if esb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if esb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	if esb.EmoteSet.Name == "" {
		return errors.ErrMissingRequiredField().SetDetail("Name")
	}

	// Check actor's permissions
	actor := opt.Actor
	if !opt.SkipValidation {
		if actor.ID.IsZero() {
			return errors.ErrUnauthorized()
		}

		if !actor.HasPermission(structures.RolePermissionCreateEmoteSet) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{"MISSING_PERMISSION": "CREATE_EMOTE_SET"})
		}

		// The set being created has an owner different to the actor
		// This requires permission
		if actor.ID != esb.EmoteSet.OwnerID && !actor.HasPermission(structures.RolePermissionManageUsers) {
			noPrivilege := errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to create an Emote Set on behalf of this user")

			if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
				"_id": esb.EmoteSet.OwnerID,
			}).Decode(&esb.EmoteSet.Owner); err != nil {
				return errors.ErrUnknownUser()
			}

			ed, ok, _ := esb.EmoteSet.Owner.GetEditor(actor.ID)
			if !ok || !ed.HasPermission(structures.UserEditorPermissionManageEmoteSets) {
				return noPrivilege
			}
		}

		// The user has created too many sets
		if count, _ := m.mongo.Collection(mongo.CollectionNameEmoteSets).CountDocuments(ctx, bson.M{
			"owner_id": esb.EmoteSet.OwnerID,
		}); count > MAXIMUM_ALLOWED_EMOTE_SETS {
			return errors.ErrInsufficientPrivilege().SetDetail("You've reached the limit for Emote Sets (%d)", MAXIMUM_ALLOWED_EMOTE_SETS)
		}

		// The emote set's name is not valid
		if err := esb.EmoteSet.Validator().Name(); err != nil {
			return err
		}
	}

	// Create the emote set
	esb.EmoteSet.ID = primitive.NewObjectIDFromTimestamp(time.Now())

	result, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).InsertOne(ctx, esb.EmoteSet)
	if err != nil {
		return err
	}

	// Get the newly created emote set
	if err = m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&esb.EmoteSet); err != nil {
		return err
	}

	// Write audit log
	alb := structures.NewAuditLogBuilder(structures.AuditLog{
		Changes: []*structures.AuditLogChange{},
	}).
		SetKind(structures.AuditLogKindCreateEmoteSet).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmoteSet).
		SetTargetID(esb.EmoteSet.ID)

	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("failed to write audit log", "error", err)
	}

	esb.MarkAsTainted()

	return nil
}

type EmoteSetMutationOptions struct {
	Actor          structures.User
	SkipValidation bool
}

type EmoteSetMutationSetEmoteOptions struct {
	Actor    structures.User
	Emotes   []EmoteSetMutationSetEmoteItem
	Channels []primitive.ObjectID
}

type EmoteSetMutationSetEmoteItem struct {
	Action structures.ListItemAction
	ID     primitive.ObjectID
	Name   string
	Flags  structures.ActiveEmoteFlag

	emote *structures.Emote
}
