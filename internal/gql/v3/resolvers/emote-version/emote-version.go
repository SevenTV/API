package emoteversion

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/api/internal/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.EmoteVersionResolver {
	return &Resolver{r}
}

func (*Resolver) Images(ctx context.Context, obj *model.EmoteVersion, format []model.ImageFormat) ([]*model.Image, error) {
	return helpers.FilterImages(obj.Images, format), nil
}
