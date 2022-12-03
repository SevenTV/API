package middleware

import (
	"strings"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/middleware"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
)

func Auth(gCtx global.Context, required bool) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		// Parse token from header
		h := utils.B2S(ctx.Request.Header.Peek("Authorization"))
		s := strings.Split(h, "Bearer ")

		if len(s) != 2 {
			if !required {
				return nil
			}

			return errors.ErrUnauthorized().SetFields(errors.Fields{"message": "Bad Authorization Header"})
		}

		t := s[1]

		// Verify the token
		user, err := middleware.DoAuth(gCtx, t)
		if err != nil {
			if !required {
				return nil
			}

			return err
		}

		ctx.SetActor(user)

		return nil
	}
}
