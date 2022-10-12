package model

import (
	"strconv"

	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleModel struct {
	ID        primitive.ObjectID `json:"id"`
	Name      string             `json:"name"`
	Position  int32              `json:"position"`
	Color     int32              `json:"color"`
	Allowed   string             `json:"allowed"`
	Denied    string             `json:"denied"`
	Invisible bool               `json:"invisible,omitempty" extensions:"x-omitempty"`
}

func (x *modelizer) Role(v structures.Role) RoleModel {
	return RoleModel{
		ID:        v.ID,
		Name:      v.Name,
		Position:  v.Position,
		Color:     int32(v.Color),
		Allowed:   strconv.Itoa(int(v.Allowed)),
		Denied:    strconv.Itoa(int(v.Denied)),
		Invisible: v.Invisible,
	}
}
