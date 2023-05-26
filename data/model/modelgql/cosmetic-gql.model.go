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
		ID:    xm.ID,
		Name:  xm.Name,
		Color: color,
		Shadows: utils.Map(xm.Shadows, func(x model.CosmeticPaintShadow) *gql_model.CosmeticPaintShadow {
			return &gql_model.CosmeticPaintShadow{
				XOffset: x.OffsetX,
				YOffset: x.OffsetY,
				Radius:  x.Radius,
				Color:   int(x.Color),
			}
		}),
		Gradients: utils.Map(xm.Gradients, func(x model.CosmeticPaintGradient) *gql_model.CosmeticPaintGradient {
			return &gql_model.CosmeticPaintGradient{
				Function:     gql_model.CosmeticPaintFunction(x.Function),
				CanvasRepeat: string(x.CanvasRepeat),
				Size:         x.Size[:],
				Stops: utils.Map(x.Stops, func(x model.CosmeticPaintGradientStop) *gql_model.CosmeticPaintStop {
					return &gql_model.CosmeticPaintStop{
						At:       x.At,
						Color:    int(x.Color),
						CenterAt: x.CenterAt[:],
					}
				}),
				Angle:    int(x.Angle),
				Repeat:   x.Repeat,
				ImageURL: &x.ImageURL,
				Shape:    &x.Shape,
				At:       x.At[:],
			}
		}),
		Flairs: utils.Map(xm.Flairs, func(x model.CosmeticPaintFlair) *gql_model.CosmeticPaintFlair {
			return &gql_model.CosmeticPaintFlair{
				Kind:    gql_model.CosmeticPaintFlairKind(x.Kind),
				XOffset: x.OffsetX,
				YOffset: x.OffsetY,
				Width:   x.Width,
				Height:  x.Height,
				Data:    x.Data,
			}
		}),
		Text: func() *gql_model.CosmeticPaintText {
			if xm.Text == nil {
				return nil
			}

			return &gql_model.CosmeticPaintText{
				Weight: utils.PointerOf(int(xm.Text.Weight)),
				Shadows: utils.Map(xm.Text.Shadows, func(x model.CosmeticPaintShadow) *gql_model.CosmeticPaintShadow {
					return &gql_model.CosmeticPaintShadow{
						XOffset: x.OffsetX,
						YOffset: x.OffsetY,
						Radius:  x.Radius,
						Color:   int(x.Color),
					}
				}),
				Transform: utils.PointerOf(string(xm.Text.Transform)),
				Stroke: func() *gql_model.CosmeticPaintStroke {
					if xm.Text.Stroke == nil {
						return nil
					}

					return &gql_model.CosmeticPaintStroke{
						Color: int(xm.Text.Stroke.Color),
						Width: xm.Text.Stroke.Width,
					}
				}(),
				Variant: new(string),
			}
		}(),
		Function: gql_model.CosmeticPaintFunction(xm.Function),
		Repeat:   xm.Repeat,
		Angle:    int(xm.Angle),
		Shape:    utils.Ternary(xm.Shape != "", &xm.Shape, nil),
		ImageURL: utils.Ternary(xm.ImageURL != "", &xm.ImageURL, nil),
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
