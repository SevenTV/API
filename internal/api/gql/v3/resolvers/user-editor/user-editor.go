package user_editor

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/structures/v3"
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

	u, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.User(u).ToPartial().GQL(), nil
}
