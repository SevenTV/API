package model

import (
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PresenceModel struct {
	ID        primitive.ObjectID `json:"id"`
	UserID    primitive.ObjectID `json:"user_id"`
	Timestamp int64              `json:"timestamp"`
	TTL       int64              `json:"ttl"`
	Kind      PresenceKind       `json:"kind"`
}

type PresenceKind uint8

const (
	UserPresenceKindUnknown PresenceKind = iota
	UserPresenceKindChannel
	UserPresenceKindWebPage
)

func (m *modelizer) Presence(v structures.UserPresence[bson.Raw]) PresenceModel {
	return PresenceModel{
		ID:        v.ID,
		UserID:    v.UserID,
		Timestamp: v.Timestamp.UnixMilli(),
		TTL:       v.TTL.UnixMilli(),
		Kind:      PresenceKind(v.Kind),
	}
}

type UserPresenceWriteResponse struct {
	OK       bool          `json:"ok"`
	Presence PresenceModel `json:"presence"`
}
