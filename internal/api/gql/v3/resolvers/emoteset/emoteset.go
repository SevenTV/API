package emoteset

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.EmoteSetResolver {
	return &Resolver{r}
}

func (r *Resolver) Owner(ctx context.Context, obj *model.EmoteSet) (*model.UserPartial, error) {
	if obj.OwnerID == nil {
		return nil, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(*obj.OwnerID)
	if err != nil {
		return nil, err
	}

	return modelgql.UserPartialModel(r.Ctx.Inst().Modelizer.User(user).ToPartial()), nil
}

func (*Resolver) Emotes(ctx context.Context, obj *model.EmoteSet, limit *int, origins *bool) ([]*model.ActiveEmote, error) {
	// remove foreign emotes?
	cut := len(obj.Emotes)

	if origins != nil && !*origins {
		for i, e := range obj.Emotes {
			if !e.OriginID.IsZero() {
				cut = i
			}
		}
	}

	emotes := make([]*model.ActiveEmote, cut)
	copy(emotes, obj.Emotes[:cut])

	if limit != nil && *limit < len(emotes) {
		emotes = emotes[:*limit]
	}

	return emotes, nil
}
