package mutation

import (
	"context"

	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (*model.UserOps, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(id)
	if err != nil {
		return nil, err
	}

	m := modelgql.UserModel(r.Ctx.Inst().Modelizer.User(user))

	return &model.UserOps{
		ID:          m.ID,
		Connections: m.Connections,
	}, nil
}
