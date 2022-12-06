package helpers

import (
	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
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
