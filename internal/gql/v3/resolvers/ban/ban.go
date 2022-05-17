package ban

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

func New(r types.Resolver) generated.BanResolver {
	return &Resolver{r}
}

func (r *Resolver) Victim(ctx context.Context, obj *model.Ban) (*model.User, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.VictimID)
	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}

func (r *Resolver) Actor(ctx context.Context, obj *model.Ban) (*model.User, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.ActorID)
	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}
