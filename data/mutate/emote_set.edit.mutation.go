package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/events"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (m *Mutate) UpdateEmoteSet(ctx context.Context, esb *structures.EmoteSetBuilder, opt EmoteSetMutationOptions) error {
	if esb == nil || esb.EmoteSet.ID.IsZero() {
		return errors.ErrInternalIncompleteMutation()
	}

	actor := opt.Actor
	if !opt.SkipValidation {
		if actor.ID.IsZero() {
			return errors.ErrUnauthorized()
		}

		if !actor.HasPermission(structures.RolePermissionEditEmoteSet) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{"MISSING_PERMISSION": "EDIT_EMOTE_SET"})
		}

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

	changeFields := make([]events.ChangeField, 0)
	init := esb.Initial()

	// Update name
	if init.Name != esb.EmoteSet.Name {
		if err := esb.EmoteSet.Validator().Name(); err != nil {
			return err
		}

		changeFields = append(changeFields, events.ChangeField{
			Key:      "name",
			Type:     events.ChangeFieldTypeString,
			OldValue: init.Name,
			Value:    esb.EmoteSet.Name,
		})
	}

	if init.Privileged != esb.EmoteSet.Privileged {
		if !actor.HasPermission(structures.RolePermissionSuperAdministrator) {
			return errors.ErrInsufficientPrivilege().SetDetail("You cannot modify an emote set's privileged state")
		}

		changeFields = append(changeFields, events.ChangeField{
			Key:      "privileged",
			Type:     events.ChangeFieldTypeBool,
			OldValue: init.Privileged,
			Value:    esb.EmoteSet.Privileged,
		})
	}

	if init.OwnerID != esb.EmoteSet.OwnerID {
		if !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
			return errors.ErrInsufficientPrivilege().SetDetail("You cannot modify an emote set's owner")
		}

		changeFields = append(changeFields, events.ChangeField{
			Key:      "owner_id",
			Type:     events.ChangeFieldTypeString,
			OldValue: init.OwnerID.Hex(),
			Value:    esb.EmoteSet.OwnerID.Hex(),
		})
	}

	if init.Capacity != esb.EmoteSet.Capacity {
		var maxCapacity int32

		for _, c := range esb.EmoteSet.Owner.Connections {
			if c.EmoteSlots > maxCapacity {
				maxCapacity = c.EmoteSlots
			}
		}

		if esb.EmoteSet.Capacity > maxCapacity {
			return errors.ErrInsufficientPrivilege().SetDetail("Capacity cannot be higher than %d", maxCapacity)
		}

		changeFields = append(changeFields, events.ChangeField{
			Key:      "capacity",
			Type:     events.ChangeFieldTypeNumber,
			OldValue: init.Capacity,
			Value:    esb.EmoteSet.Capacity,
		})
	}

	if len(changeFields) > 0 {
		// Update the document
		if err := m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOneAndUpdate(
			ctx, bson.M{
				"_id": esb.EmoteSet.ID,
			},
			esb.Update,
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&esb.EmoteSet); err != nil {
			return errors.ErrInternalServerError().SetDetail(err.Error())
		}

		// Dispatch an event
		if err := m.events.Dispatch(ctx, events.EventTypeUpdateEmoteSet, events.ChangeMap{
			ID:      esb.EmoteSet.ID,
			Kind:    structures.ObjectKindEmoteSet,
			Actor:   m.modelizer.User(actor),
			Updated: changeFields,
		}, events.EventCondition{
			"object_id": esb.EmoteSet.ID.Hex(),
		}); err != nil {
			zap.S().Errorw("failed to dispatch event", "error", err)
		}
	} else {
		return errors.ErrNothingHappened()
	}

	esb.MarkAsTainted()

	return nil
}
