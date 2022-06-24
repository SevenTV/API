package helpers

import (
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/utils"
)

func FilterImages(images []*model.Image, formats []model.ImageFormat) []*model.Image {
	result := []*model.Image{}

	for _, im := range images {
		if !utils.Contains(formats, im.Format) {
			continue
		}

		result = append(result, im)
	}

	return result
}
