package query

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/types"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.QueryResolver {
	return &Resolver{r}
}

func (r *Resolver) Z() *zap.SugaredLogger {
	return zap.S().Named("query")
}

func (r *Resolver) Actor(ctx context.Context) (*model.User, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, nil
	}

	if len(actor.Bans) > 0 {
		ban := actor.Bans[0]

		graphql.AddError(ctx, errors.ErrBanned().SetDetail("for the reason \"%s\"", ban.Reason).SetFields(errors.Fields{
			"reason":    ban.Reason,
			"expire_at": ban.ExpireAt.Format(time.RFC3339),
		}))

		return nil, nil
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(actor.ID)
	if err != nil {
		return nil, err
	}

	return r.Ctx.Inst().Modelizer.User(user).GQL(), nil
}

func (r *Resolver) User(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	bans, err := r.Ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
		Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectMemoryHole}},
	})
	if err != nil {
		return nil, err
	}

	if _, ok := bans.MemoryHole[id]; ok {
		return nil, errors.ErrUnknownUser()
	}

	user, err := r.Ctx.Inst().Loaders.UserByID().Load(id)
	if err != nil {
		return nil, err
	}

	if user.ID.IsZero() || user.ID == structures.DeletedUser.ID {
		return nil, errors.ErrUnknownUser()
	}

	return r.Ctx.Inst().Modelizer.User(user).GQL(), nil
}

func (r *Resolver) Roles(ctx context.Context) ([]*model.Role, error) {
	roles, _ := r.Ctx.Inst().Query.Roles(ctx, bson.M{})

	result := make([]*model.Role, len(roles))
	for i, rol := range roles {
		result[i] = r.Ctx.Inst().Modelizer.Role(rol).GQL()
	}

	return result, nil
}

func (r *Resolver) Role(ctx context.Context, id primitive.ObjectID) (*model.Role, error) {
	// TODO
	return nil, nil
}

func (r *Resolver) Announcement(ctx context.Context) (string, error) {
	s, err := r.Ctx.Inst().Redis.Get(ctx, "meta:announcement")
	if err != nil {
		return "", nil
	}

	return s, nil
}

type Sort struct {
	Value string    `json:"value"`
	Order SortOrder `json:"order"`
}

type SortOrder string

var (
	SortOrderAscending  SortOrder = "ASCENDING"
	SortOrderDescending SortOrder = "DESCENDING"
)

var sortOrderMap = map[string]int32{
	string(SortOrderDescending): 1,
	string(SortOrderAscending):  -1,
}
