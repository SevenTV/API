package model

import (
	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmoteSetModel struct {
	ID         primitive.ObjectID  `json:"id"`
	Name       string              `json:"name"`
	Tags       []string            `json:"tags"`
	Immutable  bool                `json:"immutable"`
	Privileged bool                `json:"privileged"`
	Emotes     []ActiveEmoteModel  `json:"emotes,omitempty" extensions:"x-omitempty"`
	Capacity   int32               `json:"capacity"`
	ParentID   *primitive.ObjectID `json:"parent_id,omitempty"`
	Owner      *UserPartialModel   `json:"owner" extensions:"x-nullable"`
}

type ActiveEmoteModel struct {
	ID        primitive.ObjectID   `json:"id"`
	Name      string               `json:"name"`
	Flags     ActiveEmoteFlagModel `json:"flags"`
	Timestamp int64                `json:"timestamp"`
	ActorID   *primitive.ObjectID  `json:"actor_id" extensions:"x-nullable"`
	Data      *EmotePartialModel   `json:"data,omitempty" extensions:"x-nullable"`
}

type ActiveEmoteFlagModel int32

const (
	ActiveEmoteFlagModelZeroWidth                ActiveEmoteFlagModel = 1 << 0  // 1 - Emote is zero-width
	ActiveEmoteFlagModelOverrideTwitchGlobal     ActiveEmoteFlagModel = 1 << 16 // 65536 - Overrides Twitch Global emotes with the same name
	ActiveEmoteFlagModelOverrideTwitchSubscriber ActiveEmoteFlagModel = 1 << 17 // 131072 - Overrides Twitch Subscriber emotes with the same name
	ActiveEmoteFlagModelOverrideBetterTTV        ActiveEmoteFlagModel = 1 << 18 // 262144 - Overrides BetterTTV emotes with the same name
)

func (x *modelizer) EmoteSet(v structures.EmoteSet) EmoteSetModel {
	emotes := make([]ActiveEmoteModel, len(v.Emotes))
	for i, e := range v.Emotes {
		emotes[i] = x.ActiveEmote(e)
	}

	var owner *UserPartialModel

	if v.Owner != nil {
		u := x.User(*v.Owner).ToPartial()
		owner = &u
	}

	if v.Tags == nil {
		v.Tags = make([]string, 0)
	}

	return EmoteSetModel{
		ID:         v.ID,
		Name:       v.Name,
		Tags:       v.Tags,
		Immutable:  v.Immutable,
		Privileged: v.Privileged,
		Emotes:     emotes,
		Capacity:   v.Capacity,
		ParentID:   v.ParentID,
		Owner:      owner,
	}
}

func (x *modelizer) ActiveEmote(v structures.ActiveEmote) ActiveEmoteModel {
	var data *EmotePartialModel

	if v.Emote != nil {
		e := x.Emote(*v.Emote).ToPartial()
		data = &e
	}

	var actorID *primitive.ObjectID
	if !v.ActorID.IsZero() {
		actorID = &v.ActorID
	}

	return ActiveEmoteModel{
		ID:        v.ID,
		Name:      v.Name,
		Flags:     ActiveEmoteFlagModel(v.Flags),
		Timestamp: v.Timestamp.UnixMilli(),
		ActorID:   actorID,
		Data:      data,
	}
}
