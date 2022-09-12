package model

import (
	"fmt"
	"regexp"

	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var twitchPictureSizeRegExp = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

type UserModel struct {
	ID                primitive.ObjectID    `json:"id"`
	UserType          UserTypeModel         `json:"type,omitempty" enums:",BOT,SYSTEM"`
	Username          string                `json:"username"`
	ProfilePictureURL string                `json:"profile_picture_url,omitempty"`
	DisplayName       string                `json:"display_name"`
	Style             UserStyle             `json:"style"`
	Biography         string                `json:"biography,omitempty" extensions:"x-omitempty"`
	Editors           []UserEditorModel     `json:"editors,omitempty"`
	RoleIDs           []primitive.ObjectID  `json:"roles"`
	Connections       []UserConnectionModel `json:"connections"`
}

type UserPartialModel struct {
	ID          primitive.ObjectID   `json:"id"`
	UserType    UserTypeModel        `json:"type,omitempty" enums:",BOT,SYSTEM"`
	Username    string               `json:"username"`
	DisplayName string               `json:"display_name"`
	Style       UserStyle            `json:"style"`
	RoleIDs     []primitive.ObjectID `json:"roles"`
}

type UserStyle struct {
	Color int32               `json:"color"`
	Paint *CosmeticPaintModel `json:"paint" extensions:"x-nullable"`
}

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

	editors := make([]UserEditorModel, len(v.Editors))
	for i, e := range v.Editors {
		editors[i] = x.UserEditor(e)
	}

	profilePictureURL := ""
	if v.AvatarID != "" {
		profilePictureURL = fmt.Sprintf("//%s/pp/%s/%s", x.cdnURL, v.ID.Hex(), v.AvatarID)
	} else {
		for _, con := range v.Connections {
			if con.Platform == structures.UserConnectionPlatformTwitch {
				if con, err := structures.ConvertUserConnection[structures.UserConnectionDataTwitch](con); err == nil {
					profilePictureURL = twitchPictureSizeRegExp.ReplaceAllString(con.Data.ProfileImageURL[6:], "70x70")
				}
			}
		}
	}

	roleIDs := make([]primitive.ObjectID, len(v.Roles))
	for i, r := range v.Roles {
		roleIDs[i] = r.ID
	}

	style := UserStyle{
		Color: int32(v.GetHighestRole().Color),
		Paint: nil,
	}

	return UserModel{
		ID:                v.ID,
		UserType:          UserTypeModel(v.UserType),
		Username:          v.Username,
		DisplayName:       utils.Ternary(v.DisplayName != "", v.DisplayName, v.Username),
		Style:             style,
		ProfilePictureURL: profilePictureURL,
		Biography:         v.Biography,
		Editors:           editors,
		RoleIDs:           roleIDs,
		Connections:       connections,
	}
}

func (um UserModel) ToPartial() UserPartialModel {
	return UserPartialModel{
		ID:          um.ID,
		UserType:    um.UserType,
		Username:    um.Username,
		DisplayName: um.DisplayName,
		RoleIDs:     um.RoleIDs,
	}
}

type UserEditorModel struct {
	ID          primitive.ObjectID `json:"id"`
	Permissions int32              `json:"permissions"`
	Visible     bool               `json:"visible"`
	AddedAt     int64              `json:"added_at"`
}

func (x *modelizer) UserEditor(v structures.UserEditor) UserEditorModel {
	return UserEditorModel{
		ID:          v.ID,
		Permissions: int32(v.Permissions),
		Visible:     v.Visible,
		AddedAt:     v.AddedAt.UnixMilli(),
	}
}

type UserConnectionModel struct {
	ID            string                      `json:"id"`
	Platform      UserConnectionPlatformModel `json:"platform" enums:"TWITCH,YOUTUBE,DISCORD"`
	Username      string                      `json:"username"`
	DisplayName   string                      `json:"display_name"`
	LinkedAt      int64                       `json:"linked_at"`
	EmoteCapacity int32                       `json:"emote_capacity"`
	EmoteSet      *EmoteSetModel              `json:"emote_set,omitempty" extensions:"x-omitempty"`

	User *UserModel `json:"user,omitempty" extensions:"x-omitempty"`
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

	var set *EmoteSetModel

	if v.EmoteSet != nil {
		s := x.EmoteSet(*v.EmoteSet)
		set = &s
	}

	return UserConnectionModel{
		ID:            v.ID,
		Platform:      UserConnectionPlatformModel(v.Platform),
		Username:      username,
		DisplayName:   displayName,
		LinkedAt:      v.LinkedAt.UnixMilli(),
		EmoteCapacity: int32(v.EmoteSlots),
		EmoteSet:      set,
	}
}
