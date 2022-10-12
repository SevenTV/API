package emote

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
)

type ResolverPartial struct {
	types.Resolver
}

func NewPartial(r types.Resolver) generated.EmotePartialResolver {
	return &ResolverPartial{r}
}

func (r *ResolverPartial) Images(ctx context.Context, obj *model.EmotePartial, format []model.ImageFormat) ([]*model.Image, error) {
	return helpers.FilterImages(obj.Images, format), nil
}

func (r *ResolverPartial) Owner(ctx context.Context, obj *model.EmotePartial) (*model.User, error) {
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

	return r.Ctx.Inst().Modelizer.User(user).GQL(), nil
}
