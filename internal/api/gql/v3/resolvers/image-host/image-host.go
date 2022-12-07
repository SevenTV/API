package imagehost

import (
	"context"

	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/api/internal/api/gql/v3/types"
	"github.com/seventv/common/utils"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ImageHostResolver {
	return &Resolver{r}
}

func (*Resolver) Files(ctx context.Context, obj *model.ImageHost, formats []model.ImageFormat) ([]*model.Image, error) {
	if len(formats) == 0 {
		return obj.Files, nil
	}

	for i := 0; i < len(obj.Files); i++ {
		f := obj.Files[i]
		if utils.Contains(formats, f.Format) {
			continue
		}

		obj.Files = utils.SliceRemove(obj.Files, i)
		i--
	}

	return obj.Files, nil
}
