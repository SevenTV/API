package events

import (
	"github.com/seventv/common/structures/v3"
)

type BridgedCommandBody struct {
	Platform    structures.UserConnectionPlatform `json:"platform"`
	Identifiers []string                          `json:"identifiers"`
	Kinds       []structures.CosmeticKind         `json:"kinds"`
}
