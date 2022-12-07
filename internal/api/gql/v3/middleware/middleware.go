package middleware

import (
	"github.com/seventv/api/internal/api/gql/v3/gen/generated"
	"github.com/seventv/api/internal/global"
)

func New(ctx global.Context) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		HasPermissions: hasPermission(ctx),
		Internal:       internal(ctx),
	}
}
