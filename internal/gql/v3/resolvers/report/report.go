package report

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/loaders"
	"github.com/seventv/api/internal/gql/v3/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ReportResolver {
	return &Resolver{r}
}

func (r *Resolver) Reporter(ctx context.Context, obj *model.Report) (*model.User, error) {
	return loaders.For(ctx).UserByID.Load(obj.Reporter.ID)
}

func (r *Resolver) Assignees(ctx context.Context, obj *model.Report) ([]*model.User, error) {
	ids := make([]primitive.ObjectID, len(obj.Assignees))
	for i, v := range obj.Assignees {
		ids[i] = v.ID
	}

	users, errs := loaders.For(ctx).UserByID.LoadAll(ids)

	return users, multierror.Append(nil, errs...).ErrorOrNil()
}
