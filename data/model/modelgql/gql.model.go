package modelgql

import (
	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
)

func ImageFile(xm model.ImageFile) *gql_model.Image {
	return &gql_model.Image{
		Name:       xm.Name,
		Format:     gql_model.ImageFormat(xm.Format),
		Width:      int(xm.Width),
		Height:     int(xm.Height),
		FrameCount: int(xm.FrameCount),
		Size:       int(xm.Size),
	}
}

func ImageHost(xm model.ImageHost) *gql_model.ImageHost {
	var files = make([]*gql_model.Image, len(xm.Files))

	for i, f := range xm.Files {
		files[i] = ImageFile(f)
	}

	return &gql_model.ImageHost{
		URL:   xm.URL,
		Files: files,
	}
}
