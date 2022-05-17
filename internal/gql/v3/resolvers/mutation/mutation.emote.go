package mutation

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (*Resolver) Emote(ctx context.Context, id primitive.ObjectID) (*model.EmoteOps, error) {
	return &model.EmoteOps{
		ID: id,
	}, nil
}
