package model

import (
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InboxMessageModel struct {
	ID           primitive.ObjectID  `json:"id"`
	Kind         MessageKind         `json:"kind"`
	CreatedAt    int64               `json:"createdAt"`
	AuthorID     *primitive.ObjectID `json:"author_id"`
	Read         bool                `json:"read"`
	ReadAt       *int64              `json:"readAt"`
	Subject      string              `json:"subject"`
	Content      string              `json:"content"`
	Important    bool                `json:"important"`
	Starred      bool                `json:"starred"`
	Pinned       bool                `json:"pinned"`
	Placeholders map[string]string   `json:"placeholders"`
}

type ModRequestMessageModel struct {
	ID               primitive.ObjectID  `json:"id"`
	Kind             MessageKind         `json:"kind"`
	CreatedAt        int64               `json:"createdAt"`
	AuthorID         *primitive.ObjectID `json:"author_id"`
	TargetKind       int                 `json:"targetKind"`
	TargetID         primitive.ObjectID  `json:"targetID"`
	Read             bool                `json:"read"`
	Wish             string              `json:"wish"`
	ActorCountryName string              `json:"actor_country_name"`
	ActorCountryCode string              `json:"actor_country_code"`
}

type MessageKind string

const (
	MessageKindEmoteComment MessageKind = "EMOTE_COMMENT"
	MessageKindModRequest   MessageKind = "MOD_REQUEST"
	MessageKindInbox        MessageKind = "INBOX"
	MessageKindNews         MessageKind = "NEWS"
)

func (m *modelizer) InboxMessage(v structures.Message[structures.MessageDataInbox]) InboxMessageModel {
	return InboxMessageModel{
		ID:           v.ID,
		Kind:         MessageKindInbox,
		CreatedAt:    v.CreatedAt.UnixMilli(),
		AuthorID:     &v.AuthorID,
		Read:         v.Read,
		ReadAt:       nil,
		Subject:      v.Data.Subject,
		Content:      v.Data.Content,
		Important:    v.Data.Important,
		Starred:      v.Data.Starred,
		Pinned:       v.Data.Pinned,
		Placeholders: utils.Ternary(v.Data.Placeholders == nil, map[string]string{}, v.Data.Placeholders),
	}
}

func (m *modelizer) ModRequestMessage(v structures.Message[structures.MessageDataModRequest]) ModRequestMessageModel {
	return ModRequestMessageModel{
		ID:               v.ID,
		Kind:             MessageKindModRequest,
		CreatedAt:        v.CreatedAt.UnixMilli(),
		AuthorID:         &v.AuthorID,
		TargetKind:       int(v.Data.TargetKind),
		TargetID:         v.Data.TargetID,
		Read:             v.Read,
		Wish:             v.Data.Wish,
		ActorCountryName: v.Data.ActorCountryName,
		ActorCountryCode: v.Data.ActorCountryCode,
	}
}
