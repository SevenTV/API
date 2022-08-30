package mutate

import (
	"context"

	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const EDITORS_MOST_COUNT = 15

func (m *Mutate) ModifyUserEditors(ctx context.Context, ub *structures.UserBuilder, opt UserEditorsOptions) error {
	if ub == nil {
		return errors.ErrInternalIncompleteMutation()
	} else if ub.IsTainted() {
		return errors.ErrMutateTaintedObject()
	}

	// Fetch relevant data
	target := ub.User
	editor := opt.Editor

	if editor == nil {
		return errors.ErrUnknownUser()
	}

	// Check permissions
	// The actor must either be privileged, the target user, or an editor with sufficient permissions
	actor := opt.Actor
	if actor.ID != target.ID && !actor.HasPermission(structures.RolePermissionManageUsers) {
		ed, ok, _ := target.GetEditor(actor.ID)
		if !ok {
			return errors.ErrInsufficientPrivilege()
		}
		// actor is an editor of target but they must also have "Manage Editors" permission to do this
		if !ed.HasPermission(structures.UserEditorPermissionManageEditors) {
			// the actor is allowed to *remove* themselve as an editor
			if !(actor.ID == editor.ID && opt.Action == structures.ListItemActionRemove) {
				return errors.ErrInsufficientPrivilege().SetDetail("You don't have permission to manage this user's editors")
			}
		}
	}

	c := &structures.AuditLogChange{
		Format: structures.AuditLogChangeFormatArrayChange,
		Key:    "editors",
	}
	log := structures.NewAuditLogBuilder(structures.AuditLog{}).
		SetKind(structures.AuditLogKindEditUser).
		SetActor(actor.ID).
		SetTargetKind(structures.ObjectKindUser).
		SetTargetID(target.ID).
		AddChanges(c)

	switch opt.Action {
	// add editor
	case structures.ListItemActionAdd:
		if len(ub.User.Editors) >= EDITORS_MOST_COUNT {
			return errors.ErrInvalidRequest().SetDetail("You have reached the maximum amount of editors allowed (%d)", EDITORS_MOST_COUNT)
		}

		ed, _, _ := ub.User.GetEditor(editor.ID)
		if !ed.ID.IsZero() {
			return errors.ErrInvalidRequest().SetDetail("User is already an editor")
		}

		ed, _, _ = ub.AddEditor(editor.ID, opt.EditorPermissions, opt.EditorVisible)
		c.WriteArrayAdded(ed)
	case structures.ListItemActionUpdate:
		oldEd, _, _ := ub.User.GetEditor(editor.ID)
		if oldEd.ID.IsZero() {
			return errors.ErrInvalidRequest().SetDetail("User is not an editor")
		}

		if oldEd.Permissions == opt.EditorPermissions && oldEd.Visible == opt.EditorVisible {
			return nil // no change
		}

		ed, i, _ := ub.UpdateEditor(editor.ID, opt.EditorPermissions, opt.EditorVisible)

		c.WriteArrayUpdated(structures.AuditLogChangeSingleValue{
			New:      ed,
			Old:      oldEd,
			Position: int32(i),
		})
	case structures.ListItemActionRemove:
		ed, _, _ := ub.RemoveEditor(editor.ID)
		if ed.ID.IsZero() {
			return errors.ErrInvalidRequest().SetDetail("User is not an editor")
		}

		c.WriteArrayRemoved(ed)
	}

	// Write mutation
	if err := m.mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(ctx, bson.M{
		"_id": target.ID,
	}, ub.Update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&ub.User); err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}

	// Write audit log entry
	if _, err := m.mongo.Collection(mongo.CollectionNameAuditLogs).InsertOne(ctx, log.AuditLog); err != nil {
		zap.S().Errorw("mongo, failed to write audit log entry for editor changes to user",
			"user_id", ub.User.ID.Hex(),
		)
	}

	ub.MarkAsTainted()

	return nil
}

type UserEditorsOptions struct {
	Actor             *structures.User
	Editor            *structures.User
	EditorPermissions structures.UserEditorPermission
	EditorVisible     bool
	Action            structures.ListItemAction
}
