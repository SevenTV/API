package emote

import (
	"context"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/gql/v3/gen/generated"
	"github.com/seventv/api/gql/v3/gen/model"
	"github.com/seventv/api/gql/v3/loaders"
	"github.com/seventv/api/gql/v3/types"
)

type ResolverPartial struct {
	types.Resolver
}

func NewPartial(r types.Resolver) generated.EmotePartialResolver {
	return &ResolverPartial{r}
}

func (r *ResolverPartial) Images(ctx context.Context, obj *model.EmotePartial, format []model.ImageFormat) ([]*model.Image, error) {
	result := []*model.Image{}
	for _, im := range obj.Images {
		ok := len(format) == 0
		if !ok {
			for _, f := range format {
				if im.Format == f {
					result = append(result, im)
				}
			}
			continue
		}

		result = append(result, im)
	}

	return result, nil
}

func (r *ResolverPartial) Owner(ctx context.Context, obj *model.EmotePartial) (*model.User, error) {
	if obj.Owner != nil && obj.Owner.ID != structures.DeletedUser.ID {
		return obj.Owner, nil
	}
	return loaders.For(ctx).UserByID.Load(obj.OwnerID)
}
