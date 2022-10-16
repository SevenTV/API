package helpers

import (
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
)

func ReportStructureToModel(s structures.Report) *model.Report {
	assignees := make([]*model.User, len(s.AssigneeIDs))
	for i, oid := range s.AssigneeIDs {
		assignees[i] = &model.User{ID: oid}
	}

	return &model.Report{
		ID:         s.ID,
		TargetKind: int(s.TargetKind),
		TargetID:   s.TargetID,
		ActorID:    s.ActorID,
		Subject:    s.Subject,
		Body:       s.Body,
		Priority:   int(s.Priority),
		Status:     model.ReportStatus(s.Status),
		CreatedAt:  s.CreatedAt,
		Notes:      []string{},
		Assignees:  assignees,
	}
}

func BanStructureToModel(s structures.Ban) *model.Ban {
	return &model.Ban{
		ID:        s.ID,
		Reason:    s.Reason,
		Effects:   int(s.Effects),
		ExpireAt:  s.ExpireAt,
		CreatedAt: s.ID.Timestamp(),
		ActorID:   s.ActorID,
		VictimID:  s.VictimID,
	}
}

func CosmeticPaintStructureToModel(s structures.Cosmetic[structures.CosmeticDataPaint]) *model.CosmeticPaint {
	var color *int
	if s.Data.Color != nil {
		color = utils.PointerOf(int(*s.Data.Color))
	}

	stops := make([]*model.CosmeticPaintStop, len(s.Data.Stops))
	for i, sto := range s.Data.Stops {
		stops[i] = &model.CosmeticPaintStop{
			At:    sto.At,
			Color: int(sto.Color),
		}
	}

	shadows := make([]*model.CosmeticPaintShadow, len(s.Data.DropShadows))
	for i, sha := range s.Data.DropShadows {
		shadows[i] = &model.CosmeticPaintShadow{
			XOffset: sha.OffsetX,
			YOffset: sha.OffsetY,
			Radius:  sha.Radius,
			Color:   int(sha.Color),
		}
	}

	return &model.CosmeticPaint{
		ID:       s.ID,
		Kind:     model.CosmeticKind(s.Kind),
		Name:     s.Name,
		Function: model.CosmeticPaintFunction(s.Data.Function),
		Color:    color,
		Angle:    int(s.Data.Angle),
		Shape:    &s.Data.Shape,
		ImageURL: &s.Data.ImageURL,
		Repeat:   s.Data.Repeat,
		Stops:    stops,
		Shadows:  shadows,
	}
}

func CosmeticBadgeStructureToModel(s structures.Cosmetic[structures.CosmeticDataBadge]) *model.CosmeticBadge {
	return &model.CosmeticBadge{
		ID:      s.ID,
		Kind:    model.CosmeticKind(s.Kind),
		Name:    s.Name,
		Tooltip: s.Data.Tooltip,
		Tag:     s.Data.Tag,
		Images:  []*model.Image{},
	}
}
