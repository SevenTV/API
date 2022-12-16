package emote

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
)

type ResolverPartial struct {
	types.Resolver
}

func NewPartial(r types.Resolver) generated.EmotePartialResolver {
	return &ResolverPartial{r}
}

func (r *ResolverPartial) Owner(ctx context.Context, obj *model.EmotePartial) (*model.UserPartial, error) {
	if obj.Owner != nil && obj.Owner.ID != structures.DeletedUser.ID {
		return obj.Owner, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.OwnerID)
	if err != nil {
		if errors.Compare(err, errors.ErrUnknownUser()) {
			return nil, nil
		}

		return nil, err
	}

	return modelgql.UserPartialModel(r.Ctx.Inst().Modelizer.User(user).ToPartial()), nil
}
