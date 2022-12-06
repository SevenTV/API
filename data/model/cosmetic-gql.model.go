package model

import (
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/utils"
)

func (xm *CosmeticPaintModel) GQL() *model.CosmeticPaint {
	var color *int
	if xm.Color != nil {
		color = utils.PointerOf(int(*xm.Color))
	}

	return &model.CosmeticPaint{
		ID:       xm.ID,
		Name:     xm.Name,
		Function: model.CosmeticPaintFunction(xm.Function),
		Color:    color,
		Repeat:   xm.Repeat,
		Angle:    int(xm.Angle),
		Shape:    utils.Ternary(xm.Shape != "", &xm.Shape, nil),
		ImageURL: utils.Ternary(xm.ImageURL != "", &xm.ImageURL, nil),
	}
}

func (xm *CosmeticBadgeModel) GQL() *model.CosmeticBadge {
	return &model.CosmeticBadge{
		ID:      xm.ID,
		Name:    xm.Name,
		Tag:     xm.Tag,
		Tooltip: xm.Tooltip,
		Host:    xm.Host.GQL(),
	}
}
