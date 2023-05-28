package middleware

import (
	"strconv"
	"strings"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
)

var allowedHeaders = []string{
	"Content-Type",
	"Content-Length",
	"Accept-Encoding",
	"Authorization",
	"Cookie",
	"X-Emote-Data",
	"X-SevenTV-Platform",
	"X-SevenTV-Version",
}

var exposedHeaders = []string{
	"X-Access-Token",
}

func CORS(gctx global.Context) Middleware {
	return func(ctx *fasthttp.RequestCtx) errors.APIError {
		reqHost := utils.B2S(ctx.Request.Header.Peek("Origin"))

		allowCredentials := utils.Contains(gctx.Config().Http.Cookie.Whitelist, reqHost)

		ctx.Response.Header.Set("Access-Control-Allow-Credentials", strconv.FormatBool(allowCredentials))
		ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		ctx.Response.Header.Set("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE")
		ctx.Response.Header.Set("Access-Control-Allow-Origin", reqHost)
		ctx.Response.Header.Set("Vary", "Origin")

		// cache cors
		ctx.Response.Header.Set("Access-Control-Max-Age", "7200")

		return nil
	}
}
