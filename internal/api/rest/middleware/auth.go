package middleware

import (
	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
)

func Auth(gCtx global.Context, required bool) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		if _, ok := ctx.GetActor(); !ok && required {
			return errors.ErrUnauthorized().SetDetail("Sign-In Required")
		}

		return nil
	}
}
