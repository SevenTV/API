package helpers

import "github.com/seventv/api/internal/gql/v3/gen/model"

func FilterImages(images []*model.Image, format []model.ImageFormat) []*model.Image {
	result := []*model.Image{}
	for _, im := range images {
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

	return result
}
