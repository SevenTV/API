package user_emote

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

func New(r types.Resolver) generated.UserEmoteResolver {
	return &Resolver{r}
}

func (r *Resolver) Emote(ctx context.Context, obj *model.UserEmote) (*model.Emote, error) {
	return loaders.For(ctx).EmoteByID.Load(obj.Emote.ID)
}
