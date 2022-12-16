package report

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver
}

// Actor implements generated.ReportResolver
func (r *Resolver) Actor(ctx context.Context, obj *model.Report) (*model.User, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.ActorID)
	if err != nil {
		return nil, err
	}

	return modelgql.UserModel(r.Ctx.Inst().Modelizer.User(user)), nil
}

func New(r types.Resolver) generated.ReportResolver {
	return &Resolver{r}
}

func (r *Resolver) Reporter(ctx context.Context, obj *model.Report) (*model.User, error) {
	user, err := r.Ctx.Inst().Loaders.UserByID().Load(obj.Actor.ID)
	if err != nil {
		return nil, err
	}

	return modelgql.UserModel(r.Ctx.Inst().Modelizer.User(user)), nil
}

func (r *Resolver) Assignees(ctx context.Context, obj *model.Report) ([]*model.User, error) {
	ids := make([]primitive.ObjectID, len(obj.Assignees))
	for i, v := range obj.Assignees {
		ids[i] = v.ID
	}

	users, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(ids)

	err := multierror.Append(nil, errs...).ErrorOrNil()
	if err != nil {
		return nil, err
	}

	result := make([]*model.User, len(users))
	for i, v := range users {
		result[i] = modelgql.UserModel(r.Ctx.Inst().Modelizer.User(v))
	}

	return result, nil
}
