package model

import (
	"time"

	"github.com/seventv/api/internal/api/gql/v3/gen/model"
)

func (xm InboxMessageModel) GQL() *model.InboxMessage {
	return &model.InboxMessage{
		ID:           xm.ID,
		Kind:         model.MessageKind(xm.Kind),
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

func (xm ModRequestMessageModel) GQL() *model.ModRequestMessage {
	return &model.ModRequestMessage{
		ID:         xm.ID,
		Kind:       model.MessageKind(xm.Kind),
		CreatedAt:  time.UnixMilli(xm.CreatedAt),
		AuthorID:   xm.AuthorID,
		TargetKind: int(xm.TargetKind),
		TargetID:   xm.TargetID,
	}
}
