package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type PresenceModel struct {
	ID        primitive.ObjectID `json:"id"`
	Authentic bool               `json:"authentic"`
	Timestamp int64              `json:"timestamp"`
	Kind      UserPresenceKind   `json:"kind"`
}

type UserPresenceKind uint8

const (
	UserPresenceKindChannel UserPresenceKind = iota + 1
	UserPresenceKindWebPage
)
