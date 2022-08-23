package subscription

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) Emote(ctx context.Context, id primitive.ObjectID) (<-chan *model.ChangeMap, error) {
	ch := r.subscribeNext(ctx, events.EventTypeUpdateEmote, id)

	return ch, nil
}
