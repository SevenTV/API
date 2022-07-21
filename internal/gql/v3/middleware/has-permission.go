package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
)

func hasPermission(gCtx global.Context) func(ctx context.Context, obj interface{}, next graphql.Resolver, role []model.Permission) (res interface{}, err error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver, role []model.Permission) (res interface{}, err error) {
		user := auth.For(ctx)
		if user.ID.IsZero() {
			return nil, errors.ErrUnauthorized()
		}

		var perms structures.RolePermission

		for _, v := range role {
			switch v {
			case model.PermissionBypassPrivacy:
				perms |= structures.RolePermissionBypassPrivacy
			case model.PermissionCreateEmoteSet:
				perms |= structures.RolePermissionCreateEmoteSet
			case model.PermissionEditEmote:
				perms |= structures.RolePermissionEditEmoteSet
			case model.PermissionCreateEmote:
				perms |= structures.RolePermissionCreateEmote
			case model.PermissionEditAnyEmote:
				perms |= structures.RolePermissionEditAnyEmote
			case model.PermissionEditAnyEmoteSet:
				perms |= structures.RolePermissionEditAnyEmoteSet
			case model.PermissionFeatureProfilePictureAnimation:
				perms |= structures.RolePermissionFeatureProfilePictureAnimation
			case model.PermissionFeatureZerowidthEmoteType:
				perms |= structures.RolePermissionFeatureZeroWidthEmoteType
			case model.PermissionManageBans:
				perms |= structures.RolePermissionManageBans
			case model.PermissionManageCosmetics:
				perms |= structures.RolePermissionManageCosmetics
			case model.PermissionManageNews:
				perms |= structures.RolePermissionManageNews
			case model.PermissionManageReports:
				perms |= structures.RolePermissionManageReports
			case model.PermissionManageRoles:
				perms |= structures.RolePermissionManageRoles
			case model.PermissionManageStack:
				perms |= structures.RolePermissionManageStack
			case model.PermissionManageUsers:
				perms |= structures.RolePermissionManageUsers
			case model.PermissionCreateReport:
				perms |= structures.RolePermissionReportCreate
			case model.PermissionSendMessages:
				perms |= structures.RolePermissionSendMessages
			case model.PermissionSuperAdministrator:
				perms |= structures.RolePermissionSuperAdministrator
			}
		}

		if !user.HasPermission(perms) {
			return nil, errors.ErrUnauthorized()
		}

		return next(ctx)
	}
}
