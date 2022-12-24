package modelgql

import (
	"time"

	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GQL User
func UserModel(xm model.UserModel) *gql_model.User {
	editors := make([]*gql_model.UserEditor, len(xm.Editors))
	for i, e := range xm.Editors {
		editors[i] = UserEditorModel(e)
	}

	connections := make([]*gql_model.UserConnection, len(xm.Connections))
	for i, c := range xm.Connections {
		connections[i] = UserConnectionModel(c)
	}

	return &gql_model.User{
		ID:          xm.ID,
		Type:        string(xm.UserType),
		Username:    xm.Username,
		DisplayName: xm.DisplayName,
		CreatedAt:   time.UnixMilli(xm.CreatedAt),
		AvatarURL:   xm.AvatarURL,
		Biography:   xm.Biography,
		Style: &gql_model.UserStyle{
			Color: int(xm.Style.Color),
		},
		Editors:     editors,
		Roles:       xm.RoleIDs,
		Connections: connections,
		OwnedEmotes: []*gql_model.Emote{},
		Reports:     []*gql_model.Report{},
	}
}

func UserPartialModel(xm model.UserPartialModel) *gql_model.UserPartial {
	connections := make([]*gql_model.UserConnectionPartial, len(xm.Connections))
	for i, c := range xm.Connections {
		connections[i] = UserConnectionPartialModel(c)
	}

	return &gql_model.UserPartial{
		ID:          xm.ID,
		Type:        string(xm.UserType),
		Username:    xm.Username,
		DisplayName: xm.DisplayName,
		AvatarURL:   xm.AvatarURL,
		CreatedAt:   xm.ID.Timestamp(),
		Style:       UserStyle(xm.Style),
		Roles:       xm.RoleIDs,
		Connections: connections,
	}
}

// GQL UserEditor
func UserEditorModel(xm model.UserEditorModel) *gql_model.UserEditor {
	return &gql_model.UserEditor{
		ID:          xm.ID,
		Permissions: int(xm.Permissions),
		Visible:     xm.Visible,
		AddedAt:     time.UnixMilli(xm.AddedAt),
	}
}

// GQL UserConnection
func UserConnectionModel(xm model.UserConnectionModel) *gql_model.UserConnection {
	var setID *primitive.ObjectID
	if xm.EmoteSet != nil {
		setID = &xm.EmoteSet.ID
	}

	return &gql_model.UserConnection{
		ID:            xm.ID,
		Platform:      gql_model.ConnectionPlatform(xm.Platform),
		Username:      xm.Username,
		DisplayName:   xm.DisplayName,
		LinkedAt:      time.UnixMilli(xm.LinkedAt),
		EmoteCapacity: int(xm.EmoteCapacity),
		EmoteSetID:    setID,
	}
}

// GQL UserConnectionPartial
func UserConnectionPartialModel(xm model.UserConnectionPartialModel) *gql_model.UserConnectionPartial {
	return &gql_model.UserConnectionPartial{
		ID:            xm.ID,
		Platform:      gql_model.ConnectionPlatform(xm.Platform),
		Username:      xm.Username,
		DisplayName:   xm.DisplayName,
		LinkedAt:      time.UnixMilli(xm.LinkedAt),
		EmoteCapacity: int(xm.EmoteCapacity),
		EmoteSetID:    xm.EmoteSetID,
	}
}

// GQL UserStyle
func UserStyle(xm model.UserStyle) *gql_model.UserStyle {
	return &gql_model.UserStyle{
		Color: int(xm.Color),
	}
}
