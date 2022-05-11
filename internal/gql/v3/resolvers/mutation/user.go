package mutation

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/loaders"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (*model.UserOps, error) {
	user, err := loaders.For(ctx).UserByID.Load(id)
	if err != nil {
		return nil, err
	}

	return &model.UserOps{
		ID:          user.ID,
		Connections: user.Connections,
	}, nil
}
