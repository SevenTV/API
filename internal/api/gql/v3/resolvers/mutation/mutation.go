package mutation

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.MutationResolver {
	return &Resolver{r}
}

func (r *Resolver) Z() *zap.SugaredLogger {
	return zap.S().Named("mutation")
}

func (r *Resolver) SetUserRole(ctx context.Context, userID primitive.ObjectID, roleID primitive.ObjectID, action model.ListItemAction) (*model.User, error) {
	// TODO
	return nil, nil
}

// Cosmetics implements generated.MutationResolver
func (*Resolver) Cosmetics(ctx context.Context, id primitive.ObjectID) (*model.CosmeticOps, error) {
	return &model.CosmeticOps{
		ID: id,
	}, nil
}
