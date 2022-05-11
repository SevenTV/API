package query

import (
	"context"
	"strconv"
	"strings"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/seventv/api/gql/v2/gen/model"
	"github.com/seventv/api/gql/v2/helpers"
	"github.com/seventv/api/gql/v2/loaders"
	"github.com/seventv/api/gql/v3/auth"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Resolver) User(ctx context.Context, id string) (*model.User, error) {
	var (
		isMe  = id == "@me"
		model *model.User
		err   error
	)
	if primitive.IsValidObjectID(id) {
		model, err = loaders.For(ctx).UserByID.Load(id)
	} else if id == "@me" {
		// Handle @me (fetch actor)
		// this sets the queried user ID to that of the actor user
		actor := auth.For(ctx)
		if actor == nil {
			return nil, errors.ErrUnauthorized()
		}
		id = actor.ID.Hex()
		model, err = loaders.For(ctx).UserByID.Load(id)
	} else {
		// at this point we assume the query is for a username
		// (it was neither an id, or the @me label)
		model, err = loaders.For(ctx).UserByUsername.Load(strings.ToLower(id))
	}

	// Check if banned
	if !isMe && model != nil {
		userID, _ := primitive.ObjectIDFromHex(model.ID)
		bans, err := r.Ctx.Inst().Query.Bans(ctx, query.BanQueryOptions{ // remove emotes made by usersa who own nothing and are happy
			Filter: bson.M{"effects": bson.M{"$bitsAnySet": structures.BanEffectMemoryHole}},
		})
		if err != nil {
			return nil, err
		}
		if _, banned := bans.MemoryHole[userID]; banned {
			return nil, errors.ErrUnknownUser()
		}
	}
	return model, err
}

func (r *Resolver) SearchUsers(ctx context.Context, queryArg string, page *int, limit *int) ([]*model.UserPartial, error) {
	actor := auth.For(ctx)
	if actor == nil || !actor.HasPermission(structures.RolePermissionManageUsers) {
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
		result[i] = helpers.UserStructureToPartialModel(r.Ctx, helpers.UserStructureToModel(r.Ctx, u))
	}

	rctx := ctx.Value(helpers.RequestCtxKey).(*fasthttp.RequestCtx)
	if rctx != nil {
		rctx.Response.Header.Set("X-Collection-Size", strconv.Itoa(totalCount))
	}
	return result, nil
}
