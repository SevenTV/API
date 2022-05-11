package resolvers

import (
	"github.com/seventv/api/gql/v2/gen/generated"
	"github.com/seventv/api/gql/v2/resolvers/emote"
	"github.com/seventv/api/gql/v2/resolvers/mutation"
	"github.com/seventv/api/gql/v2/resolvers/query"
	"github.com/seventv/api/gql/v2/resolvers/user"
	"github.com/seventv/api/gql/v2/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ResolverRoot {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Query() generated.QueryResolver {
	return query.New(r.Resolver)
}

func (r *Resolver) Mutation() generated.MutationResolver {
	return mutation.New(r.Resolver)
}

func (r *Resolver) User() generated.UserResolver {
	return user.New(r.Resolver)
}

func (r *Resolver) UserPartial() generated.UserPartialResolver {
	return user.NewPartial(r.Resolver)
}

func (r *Resolver) Emote() generated.EmoteResolver {
	return emote.New(r.Resolver)
}
