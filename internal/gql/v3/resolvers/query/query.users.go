package query

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
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
		result[i] = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL))
	}

	return result, nil
}

func (r *Resolver) Users(ctx context.Context, queryArg string, pageArg *int, limitArg *int) ([]*model.UserPartial, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return nil, errors.ErrUnauthorized()
	}

	isManager := actor.HasPermission(structures.RolePermissionManageUsers)

	// Unprivileged users must provide a query
	if !isManager && len(queryArg) < 2 {
		return nil, errors.ErrInvalidRequest().SetDetail("query must be at least 2 characters long")
	}

	page := 1
	if pageArg != nil {
		page = *pageArg

		// Disallow unprivileged users from paginating
		// This measure is to prevent scraping
		if !isManager && page > 1 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("You are not allowed to paginate users")
		}
	}

	limit := 10
	if limitArg != nil {
		limit = *limitArg

		if !isManager && limit > 10 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("limit cannot be higher than 10")
		} else if limit > 500 {
			return nil, errors.ErrInsufficientPrivilege().SetDetail("limit cannot be higher than 500")
		}
	}

	users, _, err := r.Ctx.Inst().Query.SearchUsers(ctx, bson.M{}, query.UserSearchOptions{
		Page:  page,
		Limit: limit,
		Query: queryArg,
		Sort: map[string]interface{}{
			"state.role_position": -1,
		},
	})

	result := make([]*model.UserPartial, len(users))
	for i, u := range users {
		result[i] = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(u, r.Ctx.Config().CdnURL))
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

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), nil
}
