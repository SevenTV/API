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
	ID    primitive.ObjectID `json:"id"`
	Name  string             `json:"name"`
	Color *int32             `json:"color"`
	// The canvas size for the paint
	CanvasSize [2]float64 `json:"canvas_size" bson:"canvas_size"`
	// The repeat mode of the canvas
	CanvasRepeat CosmeticPaintCanvasRepeat `json:"canvas_repeat" bson:"canvas_repeat"`
	// A list of gradients. There may be any amount, which can be stacked onto each other
	Gradients []CosmeticPaintGradient `json:"gradients"`
	// A list of shadows. There may be any amount, which can be stacked onto each other
	Shadows []CosmeticPaintShadow `json:"shadows"`
	Flairs  []CosmeticPaintFlair  `json:"flairs"`
	Text    *CosmeticPaintText    `json:"text"`
	// use `gradients`
	Function CosmeticPaintFunction `json:"function" enums:"LINEAR_GRADIENT,RADIAL_GRADIENT,URL"`
	// use `gradients`
	Repeat bool `json:"repeat"`
	// use `gradients`
	Angle int32 `json:"angle"`
	// use `gradients`
	Shape string `json:"shape"`
	// use `gradients`
	ImageURL string `json:"image_url"`
	// use `gradients`
	Stops []CosmeticPaintGradientStop `json:"stops"`
}

type CosmeticPaintGradient struct {
	// The function used to generate the paint (i.e gradients or an image)
	Function CosmeticPaintFunction `json:"function" bson:"function"`
	// Gradient stops, a list of positions and colors
	Stops []CosmeticPaintGradientStop `json:"stops" bson:"stops"`
	// For a URL-based paint, the URL to an image
	ImageURL string `json:"image_url,omitempty" bson:"image_url,omitempty"`
	// For a radial gradient, the shape of the gradient
	Shape string `json:"shape,omitempty" bson:"shape,omitempty"`
	// The degree angle of the gradient (does not apply if function is URL)
	Angle int32 `json:"angle,omitempty" bson:"angle,omitempty"`
	// Whether or not the gradient repeats outside its original area
	Repeat bool `json:"repeat" bson:"repeat"`

	// Gradient position (X/Y % values)
	At [2]float64 `json:"at,omitempty" bson:"at,omitempty"`
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
	// the center position for the gradient. X/Y % values (for radial gradients only)
	CenterAt [2]float64 `json:"center_at,omitempty" bson:"center_at,omitempty"`
}

type CosmeticPaintCanvasRepeat string

const (
	CosmeticPaintCanvasRepeatNone   CosmeticPaintCanvasRepeat = "no-repeat"
	CosmeticPaintCanvasRepeatX      CosmeticPaintCanvasRepeat = "repeat-x"
	CosmeticPaintCanvasRepeatY      CosmeticPaintCanvasRepeat = "repeat-y"
	CosmeticPaintCanvasRepeatRevert CosmeticPaintCanvasRepeat = "revert"
	CosmeticPaintCanvasRepeatRound  CosmeticPaintCanvasRepeat = "round"
	CosmeticPaintCanvasRepeatSpace  CosmeticPaintCanvasRepeat = "space"
)

type CosmeticPaintShadow struct {
	OffsetX float64 `json:"x_offset"`
	OffsetY float64 `json:"y_offset"`
	Radius  float64 `json:"radius"`
	Color   int32   `json:"color"`
}

type CosmeticPaintText struct {
	// Weight multiplier for the text. Defaults to 9x is not specified
	Weight uint8 `json:"weight,omitempty" bson:"weight,omitempty"`
	// Shadows applied to the text
	Shadows []CosmeticPaintShadow `json:"shadows,omitempty" bson:"shadows,omitempty"`
	// Text tranformation
	Transform CosmeticPaintTextTransform `json:"transform,omitempty" bson:"transform,omitempty"`
	// Text stroke
	Stroke *CosmeticPaintStroke `json:"stroke,omitempty" bson:"stroke,omitempty"`
	// (css) font variant property. non-standard
	Variant string `json:"variant" bson:"variant"`
}

type CosmeticPaintStroke struct {
	// Stroke color
	Color int32 `json:"color" bson:"color"`
	// Stroke width
	Width float64 `json:"width" bson:"width"`
}

type CosmeticPaintTextTransform string

const (
	CosmeticPaintTextTransformUppercase CosmeticPaintTextTransform = "uppercase"
	CosmeticPaintTextTransformLowercase CosmeticPaintTextTransform = "lowercase"
)

type CosmeticPaintFlair struct {
	// The kind of sprite
	Kind CosmeticPaintFlairKind `json:"kind" bson:"kind"`
	// The X offset of the flair (%)
	OffsetX float64 `json:"x_offset" bson:"x_offset"`
	// The Y offset of the flair (%)
	OffsetY float64 `json:"y_offset" bson:"y_offset"`
	// The width of the flair
	Width float64 `json:"width" bson:"width"`
	// The height of the flair
	Height float64 `json:"height" bson:"height"`
	// Base64-encoded image or vector data
	Data string `json:"data" bson:"data"`
}

type CosmeticPaintFlairKind string

const (
	CosmeticPaintSpriteKindImage  CosmeticPaintFlairKind = "IMAGE"
	CosmeticPaintSpriteKindVector CosmeticPaintFlairKind = "VECTOR"
	CosmeticPaintSpriteKindText   CosmeticPaintFlairKind = "TEXT"
)

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
		ID:           v.ID,
		Name:         v.Name,
		Function:     CosmeticPaintFunction(v.Data.Function),
		Color:        color,
		CanvasSize:   v.Data.CanvasSize,
		CanvasRepeat: CosmeticPaintCanvasRepeat(v.Data.CanvasRepeat),
		Gradients: utils.Map(v.Data.Gradients, func(x structures.CosmeticPaintGradient) CosmeticPaintGradient {
			return CosmeticPaintGradient{
				Function: CosmeticPaintFunction(x.Function),
				Stops: utils.Map(x.Stops, func(x structures.CosmeticPaintGradientStop) CosmeticPaintGradientStop {
					return CosmeticPaintGradientStop{
						At:       x.At,
						Color:    x.Color.Sum(),
						CenterAt: x.CenterAt,
					}
				}),
				ImageURL: x.ImageURL,
				Shape:    x.Shape,
				Angle:    x.Angle,
				Repeat:   x.Repeat,
				At:       x.At,
			}
		}),
		Shadows: utils.Map(v.Data.DropShadows, func(v structures.CosmeticPaintDropShadow) CosmeticPaintShadow {
			return CosmeticPaintShadow{
				OffsetX: v.OffsetX,
				OffsetY: v.OffsetY,
				Radius:  v.Radius,
				Color:   v.Color.Sum(),
			}
		}),
		Flairs: utils.Map(v.Data.Flairs, func(v structures.CosmeticPaintFlair) CosmeticPaintFlair {
			return CosmeticPaintFlair{
				Kind:    CosmeticPaintFlairKind(v.Kind),
				OffsetX: v.OffsetX,
				OffsetY: v.OffsetY,
				Width:   v.Width,
				Height:  v.Height,
				Data:    v.Data,
			}
		}),
		Text: func() *CosmeticPaintText {
			if v.Data.Text == nil {
				return nil
			}

			return &CosmeticPaintText{
				Weight: v.Data.Text.Weight,
				Shadows: utils.Map(v.Data.Text.Shadows, func(v structures.CosmeticPaintDropShadow) CosmeticPaintShadow {
					return CosmeticPaintShadow{
						OffsetX: v.OffsetX,
						OffsetY: v.OffsetY,
						Radius:  v.Radius,
						Color:   v.Color.Sum(),
					}
				}),
				Transform: CosmeticPaintTextTransform(v.Data.Text.Transform),
				Stroke: func() *CosmeticPaintStroke {
					if v.Data.Text.Stroke == nil {
						return nil
					}

					return &CosmeticPaintStroke{
						Color: v.Data.Text.Stroke.Color.Sum(),
						Width: v.Data.Text.Stroke.Width,
					}
				}(),
				Variant: v.Data.Text.Variant,
			}
		}(),
		Stops: utils.Map(v.Data.Stops, func(v structures.CosmeticPaintGradientStop) CosmeticPaintGradientStop {
			return CosmeticPaintGradientStop{
				At:    v.At,
				Color: v.Color.Sum(),
			}
		}),
		Repeat:   v.Data.Repeat,
		Angle:    v.Data.Angle,
		Shape:    v.Data.Shape,
		ImageURL: v.Data.ImageURL,
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
