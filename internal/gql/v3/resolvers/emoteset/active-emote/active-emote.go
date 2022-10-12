package activeemote

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ActiveEmoteResolver {
	return &Resolver{r}
}

func (r *Resolver) Actor(ctx context.Context, obj *model.ActiveEmote) (*model.UserPartial, error) {
	if obj.Actor == nil {
		return nil, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.Actor.ID)
	if err != nil {
		if errors.Compare(err, errors.ErrUnknownUser()) {
			return nil, nil
		}

		return nil, err
	}

	return r.Ctx.Inst().Modelizer.User(user).PartialGQL(), nil
}
