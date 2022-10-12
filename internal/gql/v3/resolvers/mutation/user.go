package mutation

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (*model.UserOps, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(id)
	if err != nil {
		return nil, err
	}

	m := r.Ctx.Inst().Modelizer.User(user).GQL()

	return &model.UserOps{
		ID:          m.ID,
		Connections: m.Connections,
	}, nil
}
