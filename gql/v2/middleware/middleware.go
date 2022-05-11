package middleware

import (
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v2/gen/generated"
)

func New(ctx global.Context) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		Internal: internal(ctx),
	}
}
