package model

import (
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EntitlementModel struct {
	ID    primitive.ObjectID `json:"id"`
	RefID primitive.ObjectID `json:"ref_id"`
	Kind  EntitlementKind    `json:"kind"`
}

type EntitlementKind string

const (
	EntitlementKindBadge    EntitlementKind = "BADGE"
	EntitlementKindPaint    EntitlementKind = "PAINT"
	EntitlementKindEmoteSet EntitlementKind = "EMOTE_SET"
)

func (m *modelizer) Entitlement(v structures.Entitlement[bson.Raw]) EntitlementModel {
	e, _ := structures.ConvertEntitlement[structures.EntitlementDataBase](v)

	return EntitlementModel{
		ID:    e.ID,
		RefID: e.Data.RefID,
		Kind:  EntitlementKind(e.Kind),
	}
}
