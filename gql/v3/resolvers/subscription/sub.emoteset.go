package subscription

import (
	"context"

	"github.com/seventv/api/gql/v3/gen/model"
	"github.com/seventv/api/gql/v3/loaders"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EmoteSet(ctx context.Context, id primitive.ObjectID, init *bool) (<-chan *model.EmoteSet, error) {
	getEmoteSet := func() *model.EmoteSet {
		set, err := loaders.For(ctx).EmoteSetByID.Load(id)
		if err != nil {
			return nil
		}
		return set
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
