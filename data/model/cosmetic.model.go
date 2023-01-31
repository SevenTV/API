package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CosmeticModel[T CosmeticPaintModel | CosmeticBadgeModel | CosmeticAvatarModel | json.RawMessage] struct {
	ID   primitive.ObjectID `json:"id"`
	Kind CosmeticKind       `json:"kind"`
	Data T                  `json:"data"`
}

type CosmeticKind string

const (
	CosmeticKindPaint  CosmeticKind = "PAINT"
	CosmeticKindBadge  CosmeticKind = "BADGE"
	CosmeticKindAvatar CosmeticKind = "AVATAR"
)

type CosmeticPaintModel struct {
	ID       primitive.ObjectID          `json:"id"`
	Name     string                      `json:"name"`
	Function CosmeticPaintFunction       `json:"function" enums:"LINEAR_GRADIENT,RADIAL_GRADIENT,URL"`
	Color    *int32                      `json:"color"`
	Repeat   bool                        `json:"repeat"`
	Angle    int32                       `json:"angle"`
	Shape    string                      `json:"shape"`
	ImageURL string                      `json:"image_url"`
	Stops    []CosmeticPaintGradientStop `json:"stops"`
	Shadows  []CosmeticPaintDropShadow   `json:"shadows"`
}

type CosmeticPaintFunction string

const (
	CosmeticPaintFunctionLinearGradient CosmeticPaintFunction = "LINEAR_GRADIENT"
	CosmeticPaintFunctionRadialGradient CosmeticPaintFunction = "RADIAL_GRADIENT"
	CosmeticPaintFunctionImageURL       CosmeticPaintFunction = "URL"
)

type CosmeticPaintGradientStop struct {
	At    float64 `json:"at"`
	Color int32   `json:"color"`
}

type CosmeticPaintDropShadow struct {
	OffsetX float64 `json:"x_offset"`
	OffsetY float64 `json:"y_offset"`
	Radius  float64 `json:"radius"`
	Color   int32   `json:"color"`
}

type CosmeticBadgeModel struct {
	ID      primitive.ObjectID `json:"id"`
	Name    string             `json:"name"`
	Tag     string             `json:"tag"`
	Tooltip string             `json:"tooltip"`
	Host    ImageHost          `json:"host"`
}

type CosmeticAvatarModel struct {
	ID   string           `json:"id"`
	User UserPartialModel `json:"user"`
	As   string           `json:"as,omitempty"`
	Host ImageHost        `json:"host"`
}

func (x *modelizer) Cosmetic(v structures.Cosmetic[bson.Raw]) CosmeticModel[json.RawMessage] {
	var d json.RawMessage

	switch v.Kind {
	case structures.CosmeticKindBadge:
		cos, err := structures.ConvertCosmetic[structures.CosmeticDataBadge](v)
		if err != nil {
			break
		}

		d = utils.ToJSON(x.Badge(cos))
	case structures.CosmeticKindNametagPaint:
		cos, err := structures.ConvertCosmetic[structures.CosmeticDataPaint](v)
		if err != nil {
			break
		}

		d = utils.ToJSON(x.Paint(cos))
	}

	return CosmeticModel[json.RawMessage]{
		ID:   v.ID,
		Kind: CosmeticKind(v.Kind),
		Data: d,
	}
}

func (x *modelizer) Paint(v structures.Cosmetic[structures.CosmeticDataPaint]) CosmeticPaintModel {
	var color *int32
	if v.Data.Color != nil {
		color = utils.PointerOf(v.Data.Color.Sum())
	}

	return CosmeticPaintModel{
		ID:       v.ID,
		Name:     v.Name,
		Function: CosmeticPaintFunction(v.Data.Function),
		Color:    color,
		Repeat:   v.Data.Repeat,
		Angle:    v.Data.Angle,
		Shape:    v.Data.Shape,
		ImageURL: v.Data.ImageURL,
		Stops: utils.Map(v.Data.Stops, func(v structures.CosmeticPaintGradientStop) CosmeticPaintGradientStop {
			return CosmeticPaintGradientStop{
				At:    v.At,
				Color: v.Color.Sum(),
			}
		}),
		Shadows: utils.Map(v.Data.DropShadows, func(v structures.CosmeticPaintDropShadow) CosmeticPaintDropShadow {
			return CosmeticPaintDropShadow{
				OffsetX: v.OffsetX,
				OffsetY: v.OffsetY,
				Radius:  v.Radius,
				Color:   v.Color.Sum(),
			}
		}),
	}
}

func (x *modelizer) Badge(v structures.Cosmetic[structures.CosmeticDataBadge]) CosmeticBadgeModel {
	host := ImageHost{
		URL: fmt.Sprintf("//%s/badge/%s", x.cdnURL, v.ID.Hex()),
		Files: []ImageFile{
			{
				Name:   "1x",
				Format: ImageFormatWEBP,
				Width:  18,
				Height: 18,
			},
			{
				Name:   "2x",
				Format: ImageFormatWEBP,
				Width:  36,
				Height: 36,
			},
			{
				Name:   "3x",
				Format: ImageFormatWEBP,
				Width:  72,
				Height: 72,
			},
		},
	}

	return CosmeticBadgeModel{
		ID:      v.ID,
		Name:    v.Name,
		Tooltip: v.Data.Tooltip,
		Tag:     v.Data.Tag,
		Host:    host,
	}
}

func (x *modelizer) Avatar(v structures.User) CosmeticModel[CosmeticAvatarModel] {
	// Minimize the user data
	usr := x.User(v).ToPartial()
	usr.AvatarURL = ""
	usr.RoleIDs = nil

	for i, con := range usr.Connections {
		con.EmoteSetID = nil

		usr.Connections[i] = con
	}

	if v.Avatar != nil {
		files := utils.Filter(v.Avatar.ImageFiles, func(fi structures.ImageFile) bool {
			return (fi.ContentType == "image/webp" || fi.ContentType == "image/avif") && !strings.HasSuffix(fi.Name, "_static")
		})

		sort.Slice(files, func(i, j int) bool {
			return files[i].Width < files[j].Width
		})

		return CosmeticModel[CosmeticAvatarModel]{
			ID:   v.Avatar.ID,
			Kind: CosmeticKindAvatar,
			Data: CosmeticAvatarModel{
				ID:   v.Avatar.ID.Hex(),
				User: usr,
				Host: ImageHost{
					URL: fmt.Sprintf("//%s/user/%s/av_%s", x.cdnURL, v.ID.Hex(), v.Avatar.ID.Hex()),
					Files: utils.Map(files, func(img structures.ImageFile) ImageFile {
						return x.Image(img)
					}),
				},
			},
		}
	} else if v.AvatarID != "" {
		return CosmeticModel[CosmeticAvatarModel]{
			ID:   v.ID,
			Kind: CosmeticKindAvatar,
			Data: CosmeticAvatarModel{
				ID:   v.AvatarID,
				User: usr,
				Host: ImageHost{
					URL: fmt.Sprintf("//%s/pp/%s", x.cdnURL, v.ID.Hex()),
					Files: []ImageFile{{
						Name:   v.AvatarID,
						Width:  128,
						Height: 128,
						Format: ImageFormatWEBP,
					}},
				},
			},
		}
	}

	return CosmeticModel[CosmeticAvatarModel]{}
}
