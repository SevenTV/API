package mutation

import (
	"github.com/seventv/api/internal/gql/v2/gen/generated"
	"github.com/seventv/api/internal/gql/v2/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.MutationResolver {
	return &Resolver{
		Resolver: r,
	}
}
