package query

import (
	"context"
	"encoding/json"

	"github.com/seventv/api/internal/gql/v2/gen/generated"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.QueryResolver {
	return &Resolver{r}
}

func (r *Resolver) Meta(ctx context.Context) (*model.Meta, error) {
	pipe := r.Ctx.Inst().Redis.Pipeline(ctx)
	announce := pipe.Get(ctx, "meta:announcement")
	feat := pipe.Get(ctx, "meta:featured_broadcast")

	_, _ = pipe.Exec(ctx)

	roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{})
	if err != nil {
		return nil, errors.ErrInternalServerError()
	}

	roleData := make([]string, len(roles))

	for i, r := range roles {
		b, _ := json.Marshal(r)

		roleData[i] = utils.B2S(b)
	}

	return &model.Meta{
		Announcement:      announce.Val(),
		FeaturedBroadcast: feat.Val(),
		Roles:             roleData,
	}, nil
}
