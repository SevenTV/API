package subscription

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EmoteSet(ctx context.Context, id primitive.ObjectID, init *bool) (<-chan *model.EmoteSet, error) {
	getEmoteSet := func() *model.EmoteSet {
		set, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(id)
		if err != nil {
			return nil
		}
		return helpers.EmoteSetStructureToModel(set, r.Ctx.Config().CdnURL)
	}

	ch := make(chan *model.EmoteSet, 1)
	if init != nil && *init {
		set := getEmoteSet()
		if set != nil {
			ch <- set
		}
	}

	go func() {
		defer close(ch)
		sub := r.subscribe(ctx, "emote_sets", id)
		for range sub {
			set := getEmoteSet()
			if set != nil {
				ch <- set
			}
		}
	}()

	return ch, nil
}
