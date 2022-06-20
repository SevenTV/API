package middleware

import (
	"github.com/seventv/common/errors"
	"github.com/valyala/fasthttp"
)

type Middleware = func(ctx *fasthttp.RequestCtx) errors.APIError
