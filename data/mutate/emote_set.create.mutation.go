package mutate

import (
	"context"
	"strconv"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
		if actor == nil {
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
	}

	// Create the emote set
	esb.EmoteSet.ID = primitive.NewObjectID()
	result, err := m.mongo.Collection(mongo.CollectionNameEmoteSets).InsertOne(ctx, esb.EmoteSet)

	if err != nil {
		return err
	}

	// Get the newly created emote set
	if err = m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&esb.EmoteSet); err != nil {
		return err
	}

	if err != nil {
		return err
	}

	esb.MarkAsTainted()

	return nil
}

// Edit: change the emote set
func (m *Mutate) EditEmoteSet(ctx context.Context, esb *structures.EmoteSetBuilder, opt EmoteSetMutationOptions) error {
	if esb == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if esb.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	// Check actor's permissions
	actor := opt.Actor
	set := esb.EmoteSet

	if actor == nil || !actor.HasPermission(structures.RolePermissionEditEmoteSet) {
		return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{"MISSING_PERMISSION": "EDIT_EMOTE_SET"})
	}

	if set.Privileged && !actor.HasPermission(structures.RolePermissionSuperAdministrator) {
		return errors.ErrInsufficientPrivilege().SetDetail("emote set is privileged")
	}

	if actor.ID != set.OwnerID && !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
		return errors.ErrInsufficientPrivilege().SetDetail("you do not own this emote set")
	}

	u := esb.Update
	if !opt.SkipValidation {
		// Change: Name
		if _, ok := u["name"]; ok {
			// TODO: use a regex to validate
			if len(set.Name) < structures.EmoteSetNameLengthLeast || len(set.Name) >= structures.EmoteSetNameLengthMost {
				return errors.ErrValidationRejected().SetFields(errors.Fields{
					"FIELD":          "Name",
					"MIN_LENGTH":     strconv.FormatInt(int64(structures.EmoteSetNameLengthLeast), 10),
					"MAX_LENGTH":     strconv.FormatInt(int64(structures.EmoteSetNameLengthMost), 10),
					"CURRENT_LENGTH": strconv.FormatInt(int64(len(set.Name)), 10),
				})
			}
		}

		// Change: Privileged
		// Must be super admin
		if _, ok := u["privileged"]; ok && !opt.Actor.HasPermission(structures.RolePermissionSuperAdministrator) {
			return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
				"FIELD":              "Privileged",
				"MISSING_PERMISSION": "SUPER_ADMINISTRATOR",
			})
		}

		// Change: owner
		// Must be the current owner, or have "edit any emote set" permission
		if _, ok := u["owner_id"]; ok && !actor.HasPermission(structures.RolePermissionEditAnyEmoteSet) {
			if actor.ID != set.OwnerID {
				return errors.ErrInsufficientPrivilege().SetFields(errors.Fields{
					"FIELD": "OwnerID",
				}).SetDetail("you do not own this emote set")
			}
		}
	}

	// Update the document
	if err := m.mongo.Collection(mongo.CollectionNameEmoteSets).FindOneAndUpdate(
		ctx, bson.M{
			"_id": set.ID,
		},
		esb.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(esb.EmoteSet); err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	esb.MarkAsTainted()

	return nil
}

type EmoteSetMutationOptions struct {
	Actor          *structures.User
	SkipValidation bool
}

type EmoteSetMutationSetEmoteOptions struct {
	Actor    *structures.User
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
