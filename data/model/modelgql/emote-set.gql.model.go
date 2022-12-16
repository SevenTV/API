package modelgql

import (
	"time"

	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func EmoteSetModel(xm model.EmoteSetModel) *gql_model.EmoteSet {
	var (
		emotes  = make([]*gql_model.ActiveEmote, len(xm.Emotes))
		ownerID *primitive.ObjectID
	)

	for i, ae := range xm.Emotes {
		emotes[i] = ActiveEmoteModel(ae)
	}

	if xm.Owner != nil {
		ownerID = &xm.Owner.ID
	}

	return &gql_model.EmoteSet{
		ID:         xm.ID,
		Name:       xm.Name,
		Tags:       xm.Tags,
		Emotes:     emotes,
		EmoteCount: int(xm.EmoteCount),
		Capacity:   int(xm.Capacity),
		Origins: utils.Map(xm.Origins, func(v model.EmoteSetOrigin) *gql_model.EmoteSetOrigin {
			return EmoteSetOrigin(v)
		}),
		OwnerID: ownerID,
	}
}

func ActiveEmoteModel(xm model.ActiveEmoteModel) *gql_model.ActiveEmote {
	var actorID primitive.ObjectID
	if xm.ActorID != nil {
		actorID = *xm.ActorID
	}

	return &gql_model.ActiveEmote{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Timestamp: time.UnixMilli(xm.Timestamp),
		Actor:     &gql_model.UserPartial{ID: actorID},
		OriginID:  xm.OriginID,
	}
}

func EmoteSetOrigin(xm model.EmoteSetOrigin) *gql_model.EmoteSetOrigin {
	return &gql_model.EmoteSetOrigin{
		ID:     xm.ID,
		Weight: int(xm.Weight),
		Slices: utils.Map(xm.Slices, func(v uint32) int {
			return int(v)
		}),
	}
}
