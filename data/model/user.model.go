package model

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var twitchPictureSizeRegExp = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

type UserModel struct {
	ID          primitive.ObjectID     `json:"id"`
	UserType    UserTypeModel          `json:"type,omitempty" enums:",BOT,SYSTEM"`
	Username    string                 `json:"username"`
	DisplayName string                 `json:"display_name"`
	CreatedAt   int64                  `json:"createdAt,omitempty"`
	AvatarURL   string                 `json:"avatar_url,omitempty"`
	Biography   string                 `json:"biography,omitempty" extensions:"x-omitempty"`
	Style       UserStyle              `json:"style"`
	EmoteSets   []EmoteSetPartialModel `json:"emote_sets,omitempty" extensions:"x-omitempty"`
	Editors     []UserEditorModel      `json:"editors,omitempty"`
	RoleIDs     []primitive.ObjectID   `json:"roles"`
	Connections []UserConnectionModel  `json:"connections,omitempty"`
}

type UserPartialModel struct {
	ID          primitive.ObjectID           `json:"id"`
	UserType    UserTypeModel                `json:"type,omitempty" enums:",BOT,SYSTEM"`
	Username    string                       `json:"username"`
	DisplayName string                       `json:"display_name"`
	AvatarURL   string                       `json:"avatar_url,omitempty"`
	Style       UserStyle                    `json:"style"`
	RoleIDs     []primitive.ObjectID         `json:"roles"`
	Connections []UserConnectionPartialModel `json:"connections"`
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
	var (
		connections = make([]UserConnectionModel, len(v.Connections))
		editors     = make([]UserEditorModel, len(v.Editors))
		avatarURL   string
	)

	for i, c := range v.Connections {
		connections[i] = x.UserConnection(c)

		if avatarURL == "" {
			switch c.Platform {
			case structures.UserConnectionPlatformTwitch:
				if con, err := structures.ConvertUserConnection[structures.UserConnectionDataTwitch](c); err == nil {
					avatarURL = twitchPictureSizeRegExp.ReplaceAllString(con.Data.ProfileImageURL[6:], "70x70")
				}
			case structures.UserConnectionPlatformYouTube:
				if con, err := structures.ConvertUserConnection[structures.UserConnectionDataYoutube](c); err == nil {
					avatarURL = con.Data.ProfileImageURL
				}
			}
		}
	}

	if v.Avatar != nil && !v.Avatar.ID.IsZero() {
		files := v.Avatar.ImageFiles
		i := 0

		for _, file := range files {
			if file.ContentType == "image/webp" {
				files[i] = file
				i++
			}
		}

		files = files[:i]

		var (
			largestStatic   structures.ImageFile
			largestAnimated structures.ImageFile
		)

		for _, file := range files {
			if file.FrameCount == 1 && !file.IsStatic() && file.Width > largestStatic.Width {
				largestStatic = file
				largestAnimated = file
			} else if file.IsStatic() && file.Width > largestStatic.Width {
				largestStatic = file
			} else if file.Width > largestAnimated.Width {
				largestAnimated = file
			}
		}

		if v.HasPermission(structures.RolePermissionFeatureProfilePictureAnimation) {
			avatarURL = largestAnimated.Key
		} else {
			avatarURL = largestStatic.Key
		}

		avatarURL = fmt.Sprintf("//%s/%s", x.cdnURL, avatarURL)
	} else if v.AvatarID != "" {
		avatarURL = fmt.Sprintf("//%s/pp/%s/%s", x.cdnURL, v.ID.Hex(), v.AvatarID)
	}

	for i, e := range v.Editors {
		editors[i] = x.UserEditor(e)
	}

	sort.Slice(v.Roles, func(i, j int) bool {
		return v.Roles[i].Position > v.Roles[j].Position
	})

	roleIDs := make([]primitive.ObjectID, len(v.Roles))
	for i, r := range v.Roles {
		roleIDs[i] = r.ID
	}

	style := UserStyle{
		Color: int32(v.GetHighestRole().Color),
		Paint: nil,
	}

	return UserModel{
		ID:          v.ID,
		UserType:    UserTypeModel(v.UserType),
		Username:    v.Username,
		DisplayName: utils.Ternary(v.DisplayName != "", v.DisplayName, v.Username),
		CreatedAt:   v.ID.Timestamp().UnixMilli(),
		Style:       style,
		AvatarURL:   avatarURL,
		Biography:   v.Biography,
		Editors:     editors,
		RoleIDs:     roleIDs,
		Connections: connections,
	}
}

func (um UserModel) ToPartial() UserPartialModel {
	connections := make([]UserConnectionPartialModel, len(um.Connections))
	for i, c := range um.Connections {
		connections[i] = c.ToPartial()
	}

	return UserPartialModel{
		ID:          um.ID,
		UserType:    um.UserType,
		Username:    um.Username,
		AvatarURL:   um.AvatarURL,
		Style:       um.Style,
		DisplayName: um.DisplayName,
		RoleIDs:     um.RoleIDs,
		Connections: connections,
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
