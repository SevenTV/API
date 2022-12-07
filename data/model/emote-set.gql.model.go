package model

import (
	"time"

	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/utils"
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
		ID:         xm.ID,
		Name:       xm.Name,
		Tags:       xm.Tags,
		Emotes:     emotes,
		EmoteCount: int(xm.EmoteCount),
		Capacity:   int(xm.Capacity),
		Origins: utils.Map(xm.Origins, func(v EmoteSetOrigin) *model.EmoteSetOrigin {
			return v.GQL()
		}),
		OwnerID: ownerID,
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

func (xm EmoteSetOrigin) GQL() *model.EmoteSetOrigin {
	return &model.EmoteSetOrigin{
		ID:     xm.ID,
		Weight: int(xm.Weight),
		Slices: utils.Map(xm.Slices, func(v uint32) int {
			return int(v)
		}),
	}
}
