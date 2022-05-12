package role

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.RoleResolver {
	return &Resolver{r}
}

func (r *Resolver) Members(ctx context.Context, obj *model.Role, page *int, limit *int) ([]*model.User, error) {
	// TODO
	return nil, nil
}
