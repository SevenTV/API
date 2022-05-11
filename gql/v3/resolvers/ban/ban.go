package ban

import (
	"context"

	"github.com/seventv/api/gql/v3/gen/generated"
	"github.com/seventv/api/gql/v3/gen/model"
	"github.com/seventv/api/gql/v3/loaders"
	"github.com/seventv/api/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.BanResolver {
	return &Resolver{r}
}

func (r *Resolver) Victim(ctx context.Context, obj *model.Ban) (*model.User, error) {
	return loaders.For(ctx).UserByID.Load(obj.VictimID)
}

func (r *Resolver) Actor(ctx context.Context, obj *model.Ban) (*model.User, error) {
	return loaders.For(ctx).UserByID.Load(obj.ActorID)
}
