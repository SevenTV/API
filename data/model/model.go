package model

import (
	"fmt"

	"github.com/seventv/common/structures/v3"
)

type Modelizer interface {
	User(v structures.User) UserModel
	EmoteSet(v structures.EmoteSet) EmoteSetModel
}

type modelizer struct {
	cdnURL     string
	websiteURL string
}

func NewInstance(opt ModelInstanceOptions) Modelizer {
	return &modelizer{
		cdnURL:     opt.CDN,
		websiteURL: opt.Website,
	}
}

type ModelInstanceOptions struct {
	CDN     string
	Website string
}

type Image struct {
	Name        string `json:"name"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
	FrameCount  int32  `json:"frame_count"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

func (x *modelizer) Image(v structures.EmoteFile) Image {
	return Image{
		Name:        v.Name,
		Width:       v.Width,
		Height:      v.Height,
		FrameCount:  v.FrameCount,
		Size:        v.Size,
		ContentType: v.ContentType,
		URL:         fmt.Sprintf("//%s/%s", x.cdnURL, v.Key),
	}
}
