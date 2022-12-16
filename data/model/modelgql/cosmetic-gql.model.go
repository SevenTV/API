package modelgql

import (
	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/utils"
)

func CosmeticPaint(xm model.CosmeticPaintModel) *gql_model.CosmeticPaint {
	var color *int
	if xm.Color != nil {
		color = utils.PointerOf(int(*xm.Color))
	}

	return &gql_model.CosmeticPaint{
		ID:       xm.ID,
		Name:     xm.Name,
		Function: gql_model.CosmeticPaintFunction(xm.Function),
		Color:    color,
		Repeat:   xm.Repeat,
		Angle:    int(xm.Angle),
		Shape:    utils.Ternary(xm.Shape != "", &xm.Shape, nil),
		ImageURL: utils.Ternary(xm.ImageURL != "", &xm.ImageURL, nil),
		Shadows: utils.Map(xm.Shadows, func(x model.CosmeticPaintDropShadow) *gql_model.CosmeticPaintShadow {
			return &gql_model.CosmeticPaintShadow{
				XOffset: x.OffsetX,
				YOffset: x.OffsetY,
				Radius:  x.Radius,
				Color:   int(x.Color),
			}
		}),
		Stops: utils.Map(xm.Stops, func(x model.CosmeticPaintGradientStop) *gql_model.CosmeticPaintStop {
			return &gql_model.CosmeticPaintStop{
				At:    x.At,
				Color: int(x.Color),
			}
		}),
	}
}

func CosmeticBadge(xm model.CosmeticBadgeModel) *gql_model.CosmeticBadge {
	return &gql_model.CosmeticBadge{
		ID:      xm.ID,
		Name:    xm.Name,
		Tag:     xm.Tag,
		Tooltip: xm.Tooltip,
		Host:    ImageHost(xm.Host),
	}
}
