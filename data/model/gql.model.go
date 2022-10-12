package model

import "github.com/seventv/api/internal/gql/v3/gen/model"

func (xm ImageFile) GQL() *model.Image {
	return &model.Image{
		Name:       xm.Name,
		Format:     model.ImageFormat(xm.Format),
		Width:      int(xm.Width),
		Height:     int(xm.Height),
		FrameCount: int(xm.FrameCount),
		Size:       int(xm.Size),
	}
}

func (xm ImageHost) GQL() *model.ImageHost {
	var files = make([]*model.Image, len(xm.Files))

	for i, f := range xm.Files {
		files[i] = f.GQL()
	}

	return &model.ImageHost{
		URL:   xm.URL,
		Files: files,
	}
}
