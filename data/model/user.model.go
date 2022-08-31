package model

import (
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserModel struct {
	ID          primitive.ObjectID    `json:"id"`
	UserType    UserTypeModel         `json:"type,omitempty" enums:",bot,system"`
	Username    string                `json:"username"`
	DisplayName string                `json:"display_name"`
	RoleIDs     []primitive.ObjectID  `json:"roles"`
	Connections []UserConnectionModel `json:"connections"`
}

// swagger:type string
type UserTypeModel string

var (
	UserTypeRegular UserTypeModel = ""
	UserTypeBot     UserTypeModel = "BOT"
	UserTypeSystem  UserTypeModel = "SYSTEM"
)

func (x *modelizer) User(v structures.User) UserModel {
	connections := make([]UserConnectionModel, len(v.Connections))
	for i, c := range v.Connections {
		connections[i] = x.UserConnection(c)
	}

	return UserModel{
		ID:          v.ID,
		UserType:    UserTypeModel(v.UserType),
		Username:    v.Username,
		DisplayName: utils.Ternary(v.DisplayName != "", v.DisplayName, v.Username),
		RoleIDs:     v.RoleIDs,
		Connections: connections,
	}
}

type UserConnectionModel struct {
	ID          string                      `json:"id"`
	Platform    UserConnectionPlatformModel `json:"platform" enums:"TWITCH,YOUTUBE,DISCORD"`
	Username    string                      `json:"username"`
	DisplayName string                      `json:"display_name"`
	LinkedAt    int64                       `json:"linked_at"`
	EmoteSetID  primitive.ObjectID          `json:"emote_set_id,omitempty"`
}

type UserConnectionPlatformModel string

var (
	UserConnectionPlatformTwitch  UserConnectionPlatformModel = "TWITCH"
	UserConnectionPlatformYouTube UserConnectionPlatformModel = "YOUTUBE"
	UserConnectionPlatformDiscord UserConnectionPlatformModel = "DISCORD"
)

func (x *modelizer) UserConnection(v structures.UserConnection[bson.Raw]) UserConnectionModel {
	var (
		displayName string
		username    string
	)

	switch v.Platform {
	case structures.UserConnectionPlatformTwitch:
		if con, err := structures.ConvertUserConnection[structures.UserConnectionDataTwitch](v); err == nil {
			displayName = con.Data.DisplayName
			username = con.Data.Login
		}
	case structures.UserConnectionPlatformYouTube:
		if con, err := structures.ConvertUserConnection[structures.UserConnectionDataYoutube](v); err == nil {
			displayName = con.Data.Title
			username = con.Data.ID
		}
	case structures.UserConnectionPlatformDiscord:
		if con, err := structures.ConvertUserConnection[structures.UserConnectionDataDiscord](v); err == nil {
			displayName = con.Data.Username
			username = con.Data.Username + "#" + con.Data.Discriminator
		}
	}

	return UserConnectionModel{
		ID:          v.ID,
		Platform:    UserConnectionPlatformModel(v.Platform),
		Username:    username,
		DisplayName: displayName,
		LinkedAt:    v.LinkedAt.UnixMilli(),
		EmoteSetID:  v.EmoteSetID,
	}
}
