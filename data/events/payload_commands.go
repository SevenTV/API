package events

import (
	"encoding/json"

	"github.com/seventv/common/structures/v3"
)

type BridgedCommandBody interface {
	json.RawMessage | CosmeticsCommandBody
}

type CosmeticsCommandBody struct {
	Platform   structures.UserConnectionPlatform `json:"platform"`
	Identifier string                            `json:"identifier"`
	Kinds      []structures.CosmeticKind         `json:"kinds"`
}
