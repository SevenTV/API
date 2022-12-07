package subscription

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) EmoteSet(ctx context.Context, id primitive.ObjectID) (<-chan *model.ChangeMap, error) {
	ch := r.subscribe(ctx, events.EventTypeUpdateEmoteSet, id)

	return ch, nil
}
