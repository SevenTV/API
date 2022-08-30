package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	if actor != nil {
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

	target := ub.User
	// Change the role
	switch opt.Action {
	case structures.ListItemActionAdd:
		ub.Update.AddToSet("role_ids", opt.Role.ID)
	case structures.ListItemActionRemove:
		ub.Update.Pull("role_ids", opt.Role.ID)
	}

	if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(
		ctx,
		bson.M{"_id": target.ID},
		ub.Update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(target); err != nil {
		return err
	}

	ub.MarkAsTainted()

	return nil
}

type SetUserRoleOptions struct {
	Role   *structures.Role
	Actor  *structures.User
	Action structures.ListItemAction
}
