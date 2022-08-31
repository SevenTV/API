package model

import (
	"github.com/seventv/common/structures/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmoteModel struct {
	ID primitive.ObjectID `json:"id"`
}

func (x *modelizer) Emote(v structures.Emote) EmoteModel {
	return EmoteModel{
		ID: v.ID,
	}
}
