package mutation

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/mutate"
	"github.com/seventv/api/internal/api/gql/v2/gen/model"
	"github.com/seventv/api/internal/api/gql/v2/helpers"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) AddChannelEditor(ctx context.Context, channelIDArg string, editorIDArg string, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	targetID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	editorID, er2 := primitive.ObjectIDFromHex(editorIDArg)

	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	target, _, err := r.doSetChannelEditor(ctx, &actor, structures.ListItemActionAdd, targetID, editorID)
	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) RemoveChannelEditor(ctx context.Context, channelIDArg string, editorIDArg string, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	targetID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	editorID, er2 := primitive.ObjectIDFromHex(editorIDArg)

	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	target, _, err := r.doSetChannelEditor(ctx, &actor, structures.ListItemActionRemove, targetID, editorID)
	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(target, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) doSetChannelEditor(
	ctx context.Context,
	actor *structures.User,
	action structures.ListItemAction,
	targetID primitive.ObjectID,
	editorID primitive.ObjectID,
) (structures.User, structures.User, error) {
	done := r.Ctx.Inst().Limiter.AwaitMutation(ctx)
	defer done()

	var (
		target structures.User
		editor structures.User
	)

	if targetID == editorID {
		return target, editor, errors.ErrDontBeSilly().SetDetail("you can't be your own editor")
	}

	users := []structures.User{}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
		"_id": bson.M{"$in": bson.A{targetID, editorID}},
	})
	if err != nil {
		return target, editor, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	if err = cur.All(ctx, &users); err != nil {
		if err == mongo.ErrNoDocuments {
			return target, editor, errors.ErrUnknownUser()
		}

		return target, editor, errors.ErrInternalServerError().SetDetail(err.Error())
	}

	for _, u := range users {
		switch u.ID {
		case targetID:
			target = u
		case editorID:
			editor = u
		}
	}

	if target.ID.IsZero() || editor.ID.IsZero() {
		return target, editor, errors.ErrUnknownUser()
	}

	ub := structures.NewUserBuilder(target)
	if err := r.Ctx.Inst().Mutate.ModifyUserEditors(ctx, ub, mutate.UserEditorsOptions{
		Actor:             actor,
		Editor:            &editor,
		EditorPermissions: structures.UserEditorPermissionModifyEmotes | structures.UserEditorPermissionManageEmoteSets,
		EditorVisible:     true,
		Action:            action,
	}); err != nil {
		return target, editor, err
	}

	return ub.User, editor, nil
}
