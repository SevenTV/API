package model

import (
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserConnectionModel struct {
	ID string `json:"id"`
	// The service of the connection.
	Platform UserConnectionPlatformModel `json:"platform" enums:"TWITCH,YOUTUBE,DISCORD"`
	// The username of the user on the platform.
	Username string `json:"username"`
	// The display name of the user on the platform.
	DisplayName string `json:"display_name"`
	// The time when the user linked this connection
	LinkedAt int64 `json:"linked_at"`
	// The maximum size of emote sets that may be bound to this connection.
	EmoteCapacity int32 `json:"emote_capacity"`
	// The ID of the emote set bound to this connection.
	EmoteSetID *primitive.ObjectID `json:"emote_set_id" extensions:"x-nullable"`
	// The emote set that is linked to this connection
	EmoteSet *EmoteSetModel `json:"emote_set" extensions:"x-nullable"`
	// A list of users active in the channel
	Presences []UserPartialModel `json:"presences,omitempty" extensions:"x-omitempty"`

	// App data for the user
	User *UserModel `json:"user,omitempty" extensions:"x-omitempty"`
}

type UserConnectionPartialModel struct {
	ID string `json:"id"`
	// The service of the connection.
	Platform UserConnectionPlatformModel `json:"platform" enums:"TWITCH,YOUTUBE,DISCORD"`
	// The username of the user on the platform.
	Username string `json:"username"`
	// The display name of the user on the platform.
	DisplayName string `json:"display_name"`
	// The time when the user linked this connection
	LinkedAt int64 `json:"linked_at"`
	// The maximum size of emote sets that may be bound to this connection.
	EmoteCapacity int32 `json:"emote_capacity"`
	// The emote set that is linked to this connection
	EmoteSetID *primitive.ObjectID `json:"emote_set_id" extensions:"x-nullable"`
}

type UserConnectionPlatformModel string

var (
	UserConnectionPlatformTwitch  UserConnectionPlatformModel = "TWITCH"
	UserConnectionPlatformYouTube UserConnectionPlatformModel = "YOUTUBE"
	UserConnectionPlatformDiscord UserConnectionPlatformModel = "DISCORD"
)

func (x *modelizer) UserConnection(v structures.UserConnection[bson.Raw]) UserConnectionModel {
	username, displayName := v.Username()

	var set *EmoteSetModel
	var setID *primitive.ObjectID

	if v.EmoteSet != nil {
		s := x.EmoteSet(*v.EmoteSet)
		set = &s
		setID = &v.EmoteSetID
	} else if !v.EmoteSetID.IsZero() {
		set = utils.PointerOf(x.EmoteSet(structures.EmoteSet{ID: v.EmoteSetID}))
		setID = &v.EmoteSetID
	}

	return UserConnectionModel{
		ID:            v.ID,
		Platform:      UserConnectionPlatformModel(v.Platform),
		Username:      username,
		DisplayName:   displayName,
		LinkedAt:      v.LinkedAt.UnixMilli(),
		EmoteCapacity: int32(v.EmoteSlots),
		EmoteSet:      set,
		EmoteSetID:    setID,
	}
}

func (ucm UserConnectionModel) ToPartial() UserConnectionPartialModel {
	var setID *primitive.ObjectID

	if ucm.EmoteSet != nil {
		setID = &ucm.EmoteSet.ID
	}

	return UserConnectionPartialModel{
		ID:            ucm.ID,
		Platform:      ucm.Platform,
		Username:      ucm.Username,
		DisplayName:   ucm.DisplayName,
		LinkedAt:      ucm.LinkedAt,
		EmoteCapacity: ucm.EmoteCapacity,
		EmoteSetID:    setID,
	}
}
