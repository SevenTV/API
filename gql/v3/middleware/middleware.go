package middleware

import (
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v3/gen/generated"
)

func New(ctx global.Context) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		HasPermissions: hasPermission(ctx),
		Internal:       internal(ctx),
	}
}
