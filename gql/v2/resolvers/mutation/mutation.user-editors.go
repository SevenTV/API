package mutation

import (
	"context"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/gql/v2/gen/model"
	"github.com/seventv/api/gql/v2/loaders"
	"github.com/seventv/api/gql/v3/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) AddChannelEditor(ctx context.Context, channelIDArg string, editorIDArg string, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	targetID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	editorID, er2 := primitive.ObjectIDFromHex(editorIDArg)
	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	if err := r.doSetChannelEditor(ctx, actor, mutations.ListItemActionAdd, targetID, editorID); err != nil {
		return nil, err
	}

	return loaders.For(ctx).UserByID.Load(targetID.Hex())
}

func (r *Resolver) RemoveChannelEditor(ctx context.Context, channelIDArg string, editorIDArg string, reason *string) (*model.User, error) {
	actor := auth.For(ctx)
	if actor == nil {
		return nil, errors.ErrUnauthorized()
	}

	targetID, er1 := primitive.ObjectIDFromHex(channelIDArg)
	editorID, er2 := primitive.ObjectIDFromHex(editorIDArg)
	if err := multierror.Append(er1, er2).ErrorOrNil(); err != nil {
		return nil, errors.ErrBadObjectID()
	}

	if err := r.doSetChannelEditor(ctx, actor, mutations.ListItemActionRemove, targetID, editorID); err != nil {
		return nil, err
	}

	return loaders.For(ctx).UserByID.Load(targetID.Hex())
}

func (r *Resolver) doSetChannelEditor(
	ctx context.Context,
	actor *structures.User,
	action mutations.ListItemAction,
	targetID primitive.ObjectID,
	editorID primitive.ObjectID,
) error {
	var target structures.User
	var editor structures.User
	users := []structures.User{}
	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
		"_id": bson.M{"$in": bson.A{targetID, editorID}},
	})
	if err != nil {
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	if err = cur.All(ctx, &users); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrUnknownUser()
		}
		return errors.ErrInternalServerError().SetDetail(err.Error())
	}
	for _, u := range users {
		switch u.ID {
		case targetID:
			target = u
		case editorID:
			editor = u
		}
	}
	if target.ID.IsZero() {
		return errors.ErrUnknownUser()
	}

	ub := structures.NewUserBuilder(target)
	if err := r.Ctx.Inst().Mutate.ModifyUserEditors(ctx, ub, mutations.UserEditorsOptions{
		Actor:             actor,
		Editor:            &editor,
		EditorPermissions: structures.UserEditorPermissionModifyEmotes,
		EditorVisible:     true,
		Action:            action,
	}); err != nil {
		return err
	}
	return nil
}
