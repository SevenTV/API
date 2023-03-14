package events

import (
	"encoding/json"

	"github.com/seventv/api/data/model"
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BridgedCommandBody interface {
	json.RawMessage | UserStateCommandBody | PresenceCommandBody
}

type UserStateCommandBody struct {
	Platform    structures.UserConnectionPlatform `json:"platform"`
	Identifiers []string                          `json:"identifiers"`
	Kinds       []structures.CosmeticKind         `json:"kinds"`
}

type PresenceCommandBody struct {
	Kind   model.PresenceKind `json:"kind"`
	Data   json.RawMessage    `json:"data"`
	UserID primitive.ObjectID `json:"user_id"`
	Self   bool               `json:"self"`
}
