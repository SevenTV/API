package user

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/mutations"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Editors implements generated.UserOpsResolver
func (r *ResolverOps) Editors(
	ctx context.Context,
	obj *model.UserOps,
	editorID primitive.ObjectID,
	data model.UserEditorUpdate,
) ([]*model.UserEditor, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	// Find the user being edited
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.ID)
	if err != nil {
		return nil, errors.ErrUnknownUser().SetDetail("Target")
	}

	// Find the editor whose editor privileges are being updated
	editor, err := r.Ctx.Inst().Loaders.UserByID().Load(editorID)
	if err != nil {
		return nil, errors.ErrUnknownUser().SetDetail("Editor")
	}

	// Get the specified editor as a current editor of user
	ed, isEditor, _ := user.GetEditor(editor.ID)

	// Get new permission bits
	var permissions structures.UserEditorPermission
	if data.Permissions != nil {
		permissions = structures.UserEditorPermission(*data.Permissions)
	} else { // no value set, use existing value
		if !isEditor {
			return nil, errors.ErrInvalidRequest().SetDetail("Provided no permission bits, but editor specified is not represented")
		}

		permissions = ed.Permissions
	}

	var visible bool
	if data.Visible != nil {
		visible = *data.Visible
	} else if isEditor { // no value set, use existing value
		visible = ed.Visible
	} else {
		visible = true
	}

	var action structures.ListItemAction
	if permissions == 0 {
		action = structures.ListItemActionRemove
	} else if isEditor {
		action = structures.ListItemActionUpdate
	} else {
		action = structures.ListItemActionAdd
	}

	// Set up mutation
	ub := structures.NewUserBuilder(user)
	if err = r.Ctx.Inst().Mutate.ModifyUserEditors(ctx, ub, mutations.UserEditorsOptions{
		Actor:             &actor,
		Editor:            &editor,
		EditorPermissions: permissions,
		EditorVisible:     visible,
		Action:            action,
	}); err != nil {
		return nil, err
	}

	// Return updated editors
	result := make([]*model.UserEditor, len(ub.User.Editors))
	for i, e := range ub.User.Editors {
		result[i] = helpers.UserEditorStructureToModel(e, r.Ctx.Config().CdnURL)
	}

	return result, nil
}
