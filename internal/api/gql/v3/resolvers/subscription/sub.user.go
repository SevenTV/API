package subscription

import (
	"context"

	"github.com/seventv/api/data/events"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) CurrentUser(ctx context.Context) (<-chan *model.ChangeMap, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, nil
	}

	ch := r.subscribe(ctx, events.EventTypeUpdateUser, actor.ID)

	return ch, nil
}

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (<-chan *model.ChangeMap, error) {
	ch := r.subscribe(ctx, events.EventTypeUpdateUser, id)

	return ch, nil
}
