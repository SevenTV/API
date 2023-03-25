package mutate

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// SetRole: add or remove a role for the user
func (m *Mutate) SetRole(ctx context.Context, ub *structures.UserBuilder, opt SetUserRoleOptions) error {
	if ub == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if ub.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	// Check for actor's permission to do this
	actor := opt.Actor
	if !actor.ID.IsZero() {
		if !actor.HasPermission(structures.RolePermissionManageRoles) {
			return errors.ErrInsufficientPrivilege()
		}

		if len(actor.Roles) == 0 {
			return errors.ErrInsufficientPrivilege()
		}

		highestRole := actor.GetHighestRole()
		if opt.Role.Position >= highestRole.Position {
			return errors.ErrInsufficientPrivilege()
		}
	}

	cm := events.ChangeMap{
		ID:    ub.User.ID,
		Kind:  structures.ObjectKindUser,
		Actor: m.modelizer.User(actor).ToPartial(),
	}

	alb := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetActor(actor.ID).
		SetKind(structures.AuditLogKindEditUser).
		SetTargetKind(structures.ObjectKindUser).
		SetTargetID(ub.User.ID)

	changes := make([]*structures.AuditLogChange, 1)

	// Change the role
	switch opt.Action {
	case structures.ListItemActionAdd:
		if utils.Contains(ub.User.RoleIDs, opt.Role.ID) {
			return nil
		}

		ub.Update.AddToSet("role_ids", opt.Role.ID)

		cm.Pushed = []events.ChangeField{{
			Key:   "role_ids",
			Index: utils.PointerOf(int32(len(ub.User.RoleIDs) + 1)),
			Type:  events.ChangeFieldTypeString,
			Value: opt.Role.ID.Hex(),
		}}

		changes[0] = structures.NewAuditChange("role_ids").WriteArrayAdded(opt.Role.ID)
	case structures.ListItemActionRemove:
		ind := -1

		for i, id := range ub.User.RoleIDs {
			if id == opt.Role.ID {
				ind = i
				break
			}
		}

		if ind == -1 {
			return nil
		}

		ub.Update.Pull("role_ids", opt.Role.ID)

		cm.Pulled = []events.ChangeField{{
			Key:   "role_ids",
			Index: utils.PointerOf(int32(ind)),
			Type:  events.ChangeFieldTypeString,
			Value: opt.Role.ID.Hex(),
		}}

		changes[0] = structures.NewAuditChange("role_ids").WriteArrayRemoved(opt.Role.ID)
	default:
		return errors.ErrInvalidRequest().SetDetail("No Action")
	}

	alb.AddChanges(changes...)

	if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(
		ctx,
		bson.M{
			"_id":      ub.User.ID,
			"username": ub.User.Username,
		},
		ub.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&ub.User); err != nil {
		return err
	}

	// Write audit log to DB
	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, alb.AuditLog); err != nil {
		zap.S().Errorw("failed to write audit log", "error", err)
	}

	// Emit a event
	m.events.Dispatch(ctx, events.EventTypeUpdateUser, cm, events.EventCondition{
		"object_id": ub.User.ID.Hex(),
	})

	ub.MarkAsTainted()

	return nil
}

type SetUserRoleOptions struct {
	Role   structures.Role
	Actor  structures.User
	Action structures.ListItemAction
}
