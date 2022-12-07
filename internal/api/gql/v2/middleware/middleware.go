package middleware

import (
	"github.com/seventv/api/internal/api/gql/v2/gen/generated"
	"github.com/seventv/api/internal/global"
)

func New(ctx global.Context) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		Internal: internal(ctx),
	}
}
