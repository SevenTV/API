package subscription

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) Emote(ctx context.Context, id primitive.ObjectID) (<-chan *model.ChangeMap, error) {
	ch := r.subscribe(ctx, events.EventTypeUpdateEmote, id)

	return ch, nil
}