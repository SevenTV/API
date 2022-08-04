package query

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"go.mongodb.org/mongo-driver/bson"
)

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
