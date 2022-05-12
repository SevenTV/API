package middleware

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/gql/v2/gen/generated"
)

func New(ctx global.Context) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		Internal: internal(ctx),
	}
}
