package model

import (
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GQL User
func (xm UserModel) GQL() *model.User {
	editors := make([]*model.UserEditor, len(xm.Editors))
	for i, e := range xm.Editors {
		editors[i] = e.GQL()
	}

	connections := make([]*model.UserConnection, len(xm.Connections))
	for i, c := range xm.Connections {
		connections[i] = c.GQL()
	}

	return &model.User{
		ID:          xm.ID,
		Type:        string(xm.UserType),
		Username:    xm.Username,
		DisplayName: xm.DisplayName,
		CreatedAt:   time.UnixMilli(xm.CreatedAt),
		AvatarURL:   xm.AvatarURL,
		Biography:   xm.Biography,
		Style: &model.UserStyle{
			Color: int(xm.Style.Color),
		},
		Editors:     editors,
		Roles:       xm.RoleIDs,
		Connections: connections,
		OwnedEmotes: []*model.Emote{},
		Reports:     []*model.Report{},
	}
}

func (xm UserPartialModel) GQL() *model.UserPartial {
	return &model.UserPartial{
		ID:          xm.ID,
		Type:        string(xm.UserType),
		Username:    xm.Username,
		DisplayName: xm.DisplayName,
		AvatarURL:   xm.AvatarURL,
		CreatedAt:   xm.ID.Timestamp(),
		Style:       xm.Style.GQL(),
		Roles:       xm.RoleIDs,
	}
}

// GQL UserEditor
func (xm UserEditorModel) GQL() *model.UserEditor {
	return &model.UserEditor{
		ID:          xm.ID,
		Permissions: int(xm.Permissions),
		Visible:     xm.Visible,
		AddedAt:     time.UnixMilli(xm.AddedAt),
	}
}

// GQL UserConnection
func (xm UserConnectionModel) GQL() *model.UserConnection {
	var setID *primitive.ObjectID
	if xm.EmoteSet != nil {
		setID = &xm.EmoteSet.ID
	}

	return &model.UserConnection{
		ID:            xm.ID,
		Platform:      model.ConnectionPlatform(xm.Platform),
		Username:      xm.Username,
		DisplayName:   xm.DisplayName,
		LinkedAt:      time.UnixMilli(xm.LinkedAt),
		EmoteCapacity: int(xm.EmoteCapacity),
		EmoteSetID:    setID,
	}
}

// GQL UserStyle
func (xm UserStyle) GQL() *model.UserStyle {
	return &model.UserStyle{
		Color: int(xm.Color),
	}
}
