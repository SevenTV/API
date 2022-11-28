package model

import (
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (xm EmoteSetModel) GQL() *model.EmoteSet {
	var (
		emotes  = make([]*model.ActiveEmote, len(xm.Emotes))
		ownerID *primitive.ObjectID
	)

	for i, ae := range xm.Emotes {
		emotes[i] = ae.GQL()
	}

	if xm.Owner != nil {
		ownerID = &xm.Owner.ID
	}

	return &model.EmoteSet{
		ID:       xm.ID,
		Name:     xm.Name,
		Tags:     xm.Tags,
		Emotes:   emotes,
		Capacity: int(xm.Capacity),
		OwnerID:  ownerID,
	}
}

func (xm ActiveEmoteModel) GQL() *model.ActiveEmote {
	var actorID primitive.ObjectID
	if xm.ActorID != nil {
		actorID = *xm.ActorID
	}

	return &model.ActiveEmote{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Timestamp: time.UnixMilli(xm.Timestamp),
		Actor:     &model.UserPartial{ID: actorID},
		OriginID:  xm.OriginID,
	}
}
