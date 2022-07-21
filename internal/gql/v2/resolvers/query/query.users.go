package query

import (
	"context"
	"strconv"
	"strings"

	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/structures/v3/query"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) User(ctx context.Context, identifier string) (*model.User, error) {
	var (
		err  error
		user structures.User
	)

	switch identifier {
	case "@me":
		// Handle @me (fetch actor)
		// this sets the queried user ID to that of the actor user
		actor := auth.For(ctx)
		if actor.ID.IsZero() {
			return nil, errors.ErrUnauthorized()
		}

		user, err = r.Ctx.Inst().Loaders.UserByID().Load(actor.ID)
	default:
		var id primitive.ObjectID
		if primitive.IsValidObjectID(identifier) {
			id, _ = primitive.ObjectIDFromHex(identifier)
		}

		filter := utils.Ternary(id.IsZero(), bson.M{"$or": bson.A{
			bson.M{"connections.id": strings.ToLower(identifier)},
			bson.M{"username": strings.ToLower(identifier)},
		}}, bson.M{
			"_id": id,
		})

		user, err = r.Ctx.Inst().Query.Users(ctx, filter).First()
		if err != nil {
			if errors.Compare(err, errors.ErrNoItems()) {
				return nil, errors.ErrUnknownUser()
			}

			return nil, err
		}

		bans, err := r.Ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
			Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectMemoryHole}},
		})
		if err != nil {
			return nil, err
		}

		if _, banned := bans.MemoryHole[user.ID]; banned {
			return nil, errors.ErrUnknownUser()
		}
	}

	if err != nil {
		return nil, err
	}

	return helpers.UserStructureToModel(user, r.Ctx.Config().CdnURL), err
}

func (r *Resolver) SearchUsers(ctx context.Context, queryArg string, page *int, limit *int) ([]*model.UserPartial, error) {
	actor := auth.For(ctx)
	if actor.ID.IsZero() || !actor.HasPermission(structures.RolePermissionManageUsers) {
		return nil, errors.ErrInsufficientPrivilege()
	}

	users, totalCount, err := r.Ctx.Inst().Query.SearchUsers(ctx, bson.M{}, query.UserSearchOptions{
		Page:  1,
		Limit: 250,
		Query: queryArg,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.UserPartial, len(users))
	for i, u := range users {
		result[i] = helpers.UserStructureToPartialModel(helpers.UserStructureToModel(u, r.Ctx.Config().CdnURL))
	}

	rctx, _ := ctx.Value(helpers.RequestCtxKey).(*fasthttp.RequestCtx)
	if rctx != nil {
		rctx.Response.Header.Set("X-Collection-Size", strconv.Itoa(totalCount))
	}

	return result, nil
}
