package user_emote

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

func New(r types.Resolver) generated.UserEmoteResolver {
	return &Resolver{r}
}

func (r *Resolver) Emote(ctx context.Context, obj *model.UserEmote) (*model.Emote, error) {
	emote, err := r.Ctx.Inst().Loaders.EmoteByID().Load(obj.Emote.ID)
	if err != nil {
		return nil, err
	}

	return helpers.EmoteStructureToModel(emote, r.Ctx.Config().CdnURL), nil
}
