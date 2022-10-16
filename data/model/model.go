package model

import (
	"fmt"
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
	ActiveEmote(v structures.ActiveEmote) ActiveEmoteModel
	Role(v structures.Role) RoleModel
	InboxMessage(v structures.Message[structures.MessageDataInbox]) InboxMessageModel
	ModRequestMessage(v structures.Message[structures.MessageDataModRequest]) ModRequestMessageModel
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
	StaticName string      `json:"static_name"`
	Width      int32       `json:"width"`
	Height     int32       `json:"height"`
	FrameCount int32       `json:"frame_count,omitempty"`
	Size       int64       `json:"size,omitempty"`
	Format     ImageFormat `json:"format"`
}

type ImageFormat string

const (
	ImageFormatAVIF ImageFormat = "AVIF"
	ImageFormatWEBP ImageFormat = "WEBP"
)

func (x *modelizer) Image(v structures.ImageFile) ImageFile {
	ext := strings.Split(v.ContentType, "/")[1]
	format := strings.ToUpper(ext)

	return ImageFile{
		Name:       fmt.Sprintf("%s.%s", v.Name, ext),
		StaticName: strings.Replace(v.Name, ".", "_static.", 1),
		Format:     ImageFormat(format),
		Width:      v.Width,
		Height:     v.Height,
		FrameCount: v.FrameCount,
		Size:       v.Size,
	}
}
