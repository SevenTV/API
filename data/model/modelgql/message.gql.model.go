package modelgql

import (
	"time"

	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
)

func InboxMessageModel(xm model.InboxMessageModel) *gql_model.InboxMessage {
	return &gql_model.InboxMessage{
		ID:           xm.ID,
		Kind:         gql_model.MessageKind(xm.Kind),
		CreatedAt:    time.UnixMilli(xm.CreatedAt),
		AuthorID:     xm.AuthorID,
		Read:         xm.Read,
		Subject:      xm.Subject,
		Content:      xm.Content,
		Important:    xm.Important,
		Starred:      xm.Starred,
		Pinned:       xm.Pinned,
		Placeholders: xm.Placeholders,
	}
}

func ModRequestMessageModel(xm model.ModRequestMessageModel) *gql_model.ModRequestMessage {
	return &gql_model.ModRequestMessage{
		ID:         xm.ID,
		Kind:       gql_model.MessageKind(xm.Kind),
		CreatedAt:  time.UnixMilli(xm.CreatedAt),
		AuthorID:   xm.AuthorID,
		TargetKind: int(xm.TargetKind),
		TargetID:   xm.TargetID,
	}
}
