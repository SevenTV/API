package model

import (
	"fmt"

	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (x *modelizer) Paint(v structures.Cosmetic[structures.CosmeticDataPaint]) *CosmeticPaintModel {
	var color *int32
	if v.Data.Color != nil {
		color = utils.PointerOf(v.Data.Color.Sum())
	}

	return &CosmeticPaintModel{
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

func (x *modelizer) Badge(v structures.Cosmetic[structures.CosmeticDataBadge]) *CosmeticBadgeModel {
	host := ImageHost{
		URL: fmt.Sprintf("//%s/badge/%s", x.cdnURL, v.ID),
		Files: []ImageFile{
			{
				Name:   "1x",
				Format: ImageFormatWEBP,
			},
			{
				Name:   "2x",
				Format: ImageFormatWEBP,
			},
			{
				Name:   "3x",
				Format: ImageFormatWEBP,
			},
		},
	}

	return &CosmeticBadgeModel{
		ID:      v.ID,
		Name:    v.Name,
		Tooltip: v.Data.Tooltip,
		Host:    host,
	}
}
