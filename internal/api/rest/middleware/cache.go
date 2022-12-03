package middleware

import (
	"fmt"
	"strings"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/utils"
)

func SetCacheControl(gCtx global.Context, maxAge int, args []string) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		ctx.Response.Header.Set("Cache-Control", fmt.Sprintf(
			"max-age=%d%s %s",
			maxAge,
			utils.Ternary(len(args) > 0, ",", ""),
			strings.Join(args, ", "),
		))

		return nil
	}
}
