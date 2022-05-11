package user

import (
	"context"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/internal/gql/v2/gen/generated"
	"github.com/seventv/api/internal/gql/v2/gen/model"
	"github.com/seventv/api/internal/gql/v2/helpers"
	"github.com/seventv/api/internal/gql/v2/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
			obj.Role = helpers.RoleStructureToModel(roles[0])
		} else {
			obj.Role = helpers.RoleStructureToModel(structures.NilRole)
		}
	}
	return obj.Role, nil
}

func (r *ResolverPartial) EmoteIds(ctx context.Context, obj *model.UserPartial) ([]string, error) {
	setID, err := primitive.ObjectIDFromHex(obj.EmoteSetID)
	if err != nil {
		return nil, errors.ErrBadObjectID()
	}

	result := []string{}
	emoteSets, err := r.Ctx.Inst().Loaders.EmoteSetByID().Load(setID)
	if err != nil {
		return result, err
	}

	for _, e := range emoteSets.Emotes {
		result = append(result, e.ID.Hex())
	}
	return result, nil
}
