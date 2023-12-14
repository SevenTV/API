package query

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/data/model/modelgql"
	"github.com/seventv/api/data/query"
	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/svc/limiter"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmotesByIDs implements generated.QueryResolver
func (r *Resolver) UsersByID(ctx context.Context, list []primitive.ObjectID) ([]*model.UserPartial, error) {
	users, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(list)
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		r.Z().Errorw("failed to load users", "error", err)

		return nil, nil
	}

	result := make([]*model.UserPartial, len(users))

	for i, user := range users {
		result[i] = modelgql.UserPartialModel(r.Ctx.Inst().Modelizer.User(user).ToPartial())
	}

	return result, nil
}

func (r *Resolver) Users(ctx context.Context, queryArg string, pageArg *int, limitArg *int) ([]*model.UserPartial, error) {
	// Rate limit
	if ok := r.Ctx.Inst().Limiter.Test(ctx, "search-users", 10, time.Second*5, limiter.TestOptions{
		Incr: 1,
	}); !ok {
		return nil, errors.ErrRateLimited()
	}

	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	isManager := actor.HasPermission(structures.RolePermissionManageUsers)

	// Temporary measure until search is optimized
	if !isManager {
		return nil, errors.ErrInsufficientPrivilege().SetDetail("Search is disabled at this time")
	}

	// Unprivileged users must provide a query
	if !isManager && len(queryArg) < 2 {
		return nil, errors.ErrInvalidRequest().SetDetail("query must be at least 2 characters long")
	}

	if pageArg != nil {
		page := *pageArg

		// Disallow unprivileged users from paginating
		// This measure is to prevent scraping
		if !isManager && page > 1 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to paginate users")
		}
	}

	limit := 25
	if limitArg != nil {
		limit = *limitArg

		if !isManager && limit > 20 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("limit cannot be higher than 25")
		} else if limit > 500 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("limit cannot be higher than 500")
		}
	}

	searchResult, err := r.Ctx.Inst().Query.SearchUsers(ctx, bson.M{}, query.UserSearchOptions{
		Limit: limit,
		Query: queryArg,
		Sort: map[string]interface{}{
			"state.role_position":         -1,
			"connections.data.view_count": -1,
		},
	})

	users, errs := r.Ctx.Inst().Loaders.UserByID().LoadAll(utils.Map(searchResult, func(v structures.User) primitive.ObjectID {
		return v.ID
	}))
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		return nil, err
	}

	result := make([]*model.UserPartial, len(users))
	for i, u := range users {
		result[i] = modelgql.UserPartialModel(r.Ctx.Inst().Modelizer.User(u).ToPartial())
	}

	return result, err
}

func (r *Resolver) UserByConnection(ctx context.Context, platform model.ConnectionPlatform, id string) (*model.User, error) {
	user, err := r.Ctx.Inst().Loaders.UserByConnectionID(structures.UserConnectionPlatform(platform)).Load(id)
	if err != nil {
		return nil, err
	}

	if user.ID.IsZero() || user.ID == structures.DeletedUser.ID {
		return nil, errors.ErrUnknownUser()
	}

	return modelgql.UserModel(r.Ctx.Inst().Modelizer.User(user)), nil
}
