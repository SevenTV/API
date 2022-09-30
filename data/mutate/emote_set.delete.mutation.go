package mutate

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func (m *Mutate) DeleteEmoteSet(ctx context.Context, esb *structures.EmoteSetBuilder, opt EmoteSetMutationOptions) error {
	if esb == nil || esb.EmoteSet.ID.IsZero() {
		return errors.ErrInternalIncompleteMutation()
	}

	// Check actor's permissions
	actor := opt.Actor
	if !opt.SkipValidation {
		if actor.ID.IsZero() {
			return errors.ErrUnauthorized()
		}

		if !actor.HasPermission(structures.RolePermissionEditEmoteSet) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{"MISSING_PERMISSION": "EDIT_EMOTE_SET"})
		}

		// Check if actor can delete this set
		if actor.ID != esb.EmoteSet.OwnerID && !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
			noPrivilege := errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to modify this Emote Set")

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
	}

	if _, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).DeleteOne(ctx, bson.M{
		"_id": esb.EmoteSet.ID,
	}); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrUnknownEmoteSet()
		}

		return errors.ErrInternalServerError()
	}

	// Emit event
	if err := m.events.Dispatch(ctx, events.EventTypeDeleteEmoteSet, events.ChangeMap{
		ID:    esb.EmoteSet.OwnerID,
		Kind:  structures.ObjectKindEmoteSet,
		Actor: m.modelizer.User(actor),
	}, events.EventCondition{
		"object_id": esb.EmoteSet.ID.Hex(),
	}); err != nil {
		zap.S().Errorw("failed to dispatch event", "error", err)
	}

	// Write audit log
	alb := structures.NewAuditLogBuilder(structures.AuditLog{
		Changes: []*structures.AuditLogChange{},
	}).
		SetKind(structures.AuditLogKindDeleteEmoteSet).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindEmoteSet).
		SetTargetID(esb.EmoteSet.ID)

	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("failed to write audit log", "error", err)
	}

	esb.MarkAsTainted()

	return nil
}
