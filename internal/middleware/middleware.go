package middleware

import (
	"github.com/SevenTV/Common/errors"
	"github.com/valyala/fasthttp"
)

type Middleware = func(ctx *fasthttp.RequestCtx) errors.APIError
