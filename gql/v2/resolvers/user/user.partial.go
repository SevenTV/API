package user

import (
	"context"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/gql/v2/gen/generated"
	"github.com/seventv/api/gql/v2/gen/model"
	"github.com/seventv/api/gql/v2/helpers"
	"github.com/seventv/api/gql/v2/loaders"
	"github.com/seventv/api/gql/v2/types"
	"go.mongodb.org/mongo-driver/bson"
)

type ResolverPartial struct {
	types.Resolver
}

func NewPartial(r types.Resolver) generated.UserPartialResolver {
	return &ResolverPartial{
		Resolver: r,
	}
}

func (r *ResolverPartial) Role(ctx context.Context, obj *model.UserPartial) (*model.Role, error) {
	if obj.Role == nil {
		// Get default role
		roles, err := r.Ctx.Inst().Query.Roles(ctx, bson.M{"default": true})
		if err == nil && len(roles) > 0 {
			obj.Role = helpers.RoleStructureToModel(r.Ctx, roles[0])
		} else {
			obj.Role = helpers.RoleStructureToModel(r.Ctx, structures.NilRole)
		}
	}
	return obj.Role, nil
}

func (*ResolverPartial) EmoteIds(ctx context.Context, obj *model.UserPartial) ([]string, error) {
	result := []string{}
	emotes, err := loaders.For(ctx).UserEmotes.Load(obj.EmoteSetID)
	if err != nil {
		return result, err
	}

	for _, e := range emotes {
		result = append(result, e.ID)
	}
	return result, nil
}
