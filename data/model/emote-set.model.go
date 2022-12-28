package model

import (
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmoteSetModel struct {
	ID         primitive.ObjectID `json:"id"`
	Name       string             `json:"name"`
	Flags      EmoteSetFlagModel  `json:"flags"`
	Tags       []string           `json:"tags"`
	Immutable  bool               `json:"immutable"`
	Privileged bool               `json:"privileged"`
	Emotes     []ActiveEmoteModel `json:"emotes,omitempty" extensions:"x-omitempty"`
	EmoteCount int                `json:"emote_count,omitempty" extensions:"x-omitempty"`
	Capacity   int32              `json:"capacity"`
	Origins    []EmoteSetOrigin   `json:"origins,omitempty" extensions:"x-omitempty"`
	Owner      *UserPartialModel  `json:"owner" extensions:"x-nullable"`
}

type EmoteSetPartialModel struct {
	ID       primitive.ObjectID `json:"id"`
	Name     string             `json:"name"`
	Flags    EmoteSetFlagModel  `json:"flags"`
	Tags     []string           `json:"tags"`
	Capacity int32              `json:"capacity"`
	Owner    *UserPartialModel  `json:"owner,omitempty" extensions:"x-nullable, x-omitempty"`
}

type EmoteSetFlagModel int32

const (
	// Set is immutable, meaning it cannot be modified
	EmoteSetFlagImmutable EmoteSetFlagModel = 1 << 0
	// Set is privileged, meaning it can only be modified by its owner
	EmoteSetFlagPrivileged EmoteSetFlagModel = 1 << 1
	// Set is personal, meaning its content can be used globally and it is subject to stricter content moderation rules
	EmoteSetFlagPersonal EmoteSetFlagModel = 1 << 2
	// Set is commercial, meaning it is sold and subject to extra rules on content ownership
	EmoteSetFlagCommercial EmoteSetFlagModel = 1 << 3
)

type ActiveEmoteModel struct {
	ID        primitive.ObjectID   `json:"id"`
	Name      string               `json:"name"`
	Flags     ActiveEmoteFlagModel `json:"flags"`
	Timestamp int64                `json:"timestamp"`
	ActorID   *primitive.ObjectID  `json:"actor_id" extensions:"x-nullable"`
	Data      *EmotePartialModel   `json:"data,omitempty" extensions:"x-nullable"`
	OriginID  *primitive.ObjectID  `json:"origin_id,omitempty" extensions:"x-omitempty"`
}

type ActiveEmoteFlagModel int32

const (
	ActiveEmoteFlagModelZeroWidth                ActiveEmoteFlagModel = 1 << 0  // 1 - Emote is zero-width
	ActiveEmoteFlagModelOverrideTwitchGlobal     ActiveEmoteFlagModel = 1 << 16 // 65536 - Overrides Twitch Global emotes with the same name
	ActiveEmoteFlagModelOverrideTwitchSubscriber ActiveEmoteFlagModel = 1 << 17 // 131072 - Overrides Twitch Subscriber emotes with the same name
	ActiveEmoteFlagModelOverrideBetterTTV        ActiveEmoteFlagModel = 1 << 18 // 262144 - Overrides BetterTTV emotes with the same name
)

type EmoteSetOrigin struct {
	ID     primitive.ObjectID `json:"id"`
	Weight int32              `json:"weight"`
	Slices []uint32           `json:"slices"`
}

func (x *modelizer) EmoteSet(v structures.EmoteSet) EmoteSetModel {
	emotes := make([]ActiveEmoteModel, len(v.Emotes))
	for i, e := range v.Emotes {
		emotes[i] = x.ActiveEmote(e)
	}

	var owner *UserPartialModel

	if v.Owner != nil {
		u := x.User(*v.Owner).ToPartial()
		u.Connections = nil // clear the connections field of emote set owners as it's not needed here

		owner = &u
	} else if !v.OwnerID.IsZero() {
		owner = &UserPartialModel{ID: v.OwnerID}
	}

	if v.Tags == nil {
		v.Tags = make([]string, 0)
	}

	return EmoteSetModel{
		ID:         v.ID,
		Name:       v.Name,
		Flags:      EmoteSetFlagModel(v.Flags),
		Tags:       v.Tags,
		Immutable:  v.Immutable,
		Privileged: v.Privileged,
		Emotes:     emotes,
		EmoteCount: len(emotes),
		Capacity:   v.Capacity,
		Origins: utils.Map(v.Origins, func(v structures.EmoteSetOrigin) EmoteSetOrigin {
			return x.EmoteSetOrigin(v)
		}),
		Owner: owner,
	}
}

func (esm EmoteSetModel) ToPartial() EmoteSetPartialModel {
	return EmoteSetPartialModel{
		ID:       esm.ID,
		Name:     esm.Name,
		Flags:    esm.Flags,
		Tags:     esm.Tags,
		Capacity: esm.Capacity,
		Owner:    esm.Owner,
	}
}

func (x *modelizer) ActiveEmote(v structures.ActiveEmote) ActiveEmoteModel {
	var data *EmotePartialModel

	if v.Emote != nil {
		// TODO: This is a workaround due to active emote flags not being implemented
		// this mirrors the emote's flags to the value in the active emote
		if v.Emote.Flags.Has(structures.EmoteFlagsZeroWidth) {
			v.Flags = v.Flags.Set(structures.ActiveEmoteFlagZeroWidth)
		}

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
		OriginID:  utils.Ternary(v.Origin.ID.IsZero(), nil, &v.Origin.ID),
	}
}

func (x *modelizer) EmoteSetOrigin(v structures.EmoteSetOrigin) EmoteSetOrigin {
	return EmoteSetOrigin{
		ID:     v.ID,
		Weight: v.Weight,
		Slices: v.Slices,
	}
}
