package model

import "github.com/seventv/api/internal/gql/v3/gen/model"

func (xm RoleModel) GQL() *model.Role {
	return &model.Role{
		ID:        xm.ID,
		Name:      xm.Name,
		Position:  int(xm.Position),
		Color:     int(xm.Color),
		Allowed:   xm.Allowed,
		Denied:    xm.Denied,
		Invisible: xm.Invisible,
	}
}
