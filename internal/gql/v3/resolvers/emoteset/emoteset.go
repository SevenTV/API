package emoteset

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.EmoteSetResolver {
	return &Resolver{r}
}

func (r *Resolver) Owner(ctx context.Context, obj *model.EmoteSet) (*model.User, error) {
	if obj.OwnerID == nil {
		return nil, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(*obj.OwnerID)
	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}

func (*Resolver) Emotes(ctx context.Context, obj *model.EmoteSet, limit *int) ([]*model.ActiveEmote, error) {
	emotes := make([]*model.ActiveEmote, len(obj.Emotes))
	copy(emotes, obj.Emotes)

	if limit != nil && *limit < len(emotes) {
		emotes = emotes[:*limit]
	}

	return emotes, nil
}
