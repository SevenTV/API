package user_editor

import (
	"context"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/loaders"
	"github.com/seventv/api/internal/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.UserEditorResolver {
	return &Resolver{r}
}

func (r *Resolver) User(ctx context.Context, obj *model.UserEditor) (*model.UserPartial, error) {
	if obj.User != nil && obj.User.ID != structures.DeletedEmote.ID {
		return obj.User, nil
	}
	u, err := loaders.For(ctx).UserByID.Load(obj.ID)
	if err != nil {
		return nil, err
	}
	return helpers.UserStructureToPartialModel(r.Ctx, u), nil
}
