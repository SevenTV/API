package model

import (
	"strings"

	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

type Modelizer interface {
	Emote(v structures.Emote) EmoteModel
	User(v structures.User) UserModel
	UserEditor(v structures.UserEditor) UserEditorModel
	UserConnection(v structures.UserConnection[bson.Raw]) UserConnectionModel
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

type ImageHost struct {
	URL   string      `json:"url"`
	Files []ImageFile `json:"files"`
}

type ImageFile struct {
	Name       string      `json:"name"`
	Width      int32       `json:"width"`
	Height     int32       `json:"height"`
	FrameCount int32       `json:"frame_count"`
	Size       int64       `json:"size"`
	Format     ImageFormat `json:"format"`
}

type ImageFormat string

const (
	ImageFormatAVIF ImageFormat = "AVIF"
	ImageFormatWEBP ImageFormat = "WEBP"
)

func (x *modelizer) Image(v structures.EmoteFile) ImageFile {
	format := strings.Split(v.ContentType, "/")[1]
	format = strings.ToUpper(format)

	return ImageFile{
		Name:       v.Name,
		Format:     ImageFormat(format),
		Width:      v.Width,
		Height:     v.Height,
		FrameCount: v.FrameCount,
		Size:       v.Size,
	}
}
