package model

import (
	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
)

type Role struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int32  `json:"position"`
	Color    int32  `json:"color"`
	Allowed  int64  `json:"allowed"`
	Denied   int64  `json:"denied"`
}

func NewRole(s structures.Role) *Role {
	var p int64

	switch s.Allowed {
	case structures.RolePermissionCreateEmote:
		p |= v2structures.RolePermissionEmoteCreate
	case structures.RolePermissionEditEmote:
		p |= v2structures.RolePermissionEmoteEditOwned
	case structures.RolePermissionEditAnyEmote:
		p |= v2structures.RolePermissionEmoteEditAll
	case structures.RolePermissionCreateReport:
		p |= v2structures.RolePermissionCreateReports
	case structures.RolePermissionManageBans:
		p |= v2structures.RolePermissionBanUsers
	case structures.RolePermissionSuperAdministrator:
		p |= v2structures.RolePermissionAdministrator
	case structures.RolePermissionManageRoles:
		p |= v2structures.RolePermissionManageRoles
	case structures.RolePermissionManageUsers:
		p |= v2structures.RolePermissionManageUsers
	case structures.RolePermissionManageStack:
		p |= v2structures.RolePermissionEditApplicationMeta
	case structures.RolePermissionManageCosmetics:
		p |= v2structures.RolePermissionManageEntitlements
	case structures.RolePermissionFeatureZeroWidthEmoteType:
		p |= v2structures.RolePermissionUseZeroWidthEmote
	case structures.RolePermissionFeatureProfilePictureAnimation:
		p |= v2structures.RolePermissionUseCustomAvatars
	}

	return &Role{
		ID:       s.ID.Hex(),
		Name:     s.Name,
		Position: s.Position,
		Color:    int32(s.Color),
		Allowed:  int64(p),
		Denied:   int64(s.Denied),
	}
}
