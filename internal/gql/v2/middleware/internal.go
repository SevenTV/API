package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
)

func internal(gCtx global.Context) func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		return nil, errors.ErrInternalField()
	}
}
