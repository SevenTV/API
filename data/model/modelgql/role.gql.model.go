package modelgql

import (
	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
)

func RoleModel(xm model.RoleModel) *gql_model.Role {
	return &gql_model.Role{
		ID:        xm.ID,
		Name:      xm.Name,
		Position:  int(xm.Position),
		Color:     int(xm.Color),
		Allowed:   xm.Allowed,
		Denied:    xm.Denied,
		Invisible: xm.Invisible,
	}
}
