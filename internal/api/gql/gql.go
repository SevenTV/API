package gql

import (
	"fmt"
	"time"

	"github.com/fasthttp/router"
	"github.com/seventv/api/internal/constant"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"

	v3 "github.com/seventv/api/internal/api/gql/v3"
	"github.com/seventv/api/internal/middleware"
	"github.com/valyala/fasthttp"
)

func New(gctx global.Context) error {
	port := gctx.Config().Http.Ports.GQL
	if port == 0 {
		port = 80
	}

	gqlv3 := v3.GqlHandlerV3(gctx)

	router := router.New()

	router.RedirectTrailingSlash = true
	v3Route := func(ctx *fasthttp.RequestCtx) {
		gqlv3(ctx)
	}

	router.GET(fmt.Sprintf("/v3%s/gql", gctx.Config().Http.VersionSuffix), v3Route)
	router.POST(fmt.Sprintf("/v3%s/gql", gctx.Config().Http.VersionSuffix), v3Route)

	doCORS := middleware.CORS(gctx)
	doAuth := middleware.Auth(gctx)

	router.HandleOPTIONS = true
	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()

			// Add client IP to context
			ip := utils.B2S(ctx.Request.Header.Peek("Cf-Connecting-IP"))
			if ip == "" {
				ip = ctx.RemoteIP().String()
			}

			ctx.SetUserValue(constant.ClientIP, ip)

			defer func() {
				if err := recover(); err != nil {
					zap.S().Errorw("panic in gql request handler",
						"panic", err,
						"status", ctx.Response.StatusCode(),
						"duration", int(time.Since(start)/time.Millisecond),
						"method", utils.B2S(ctx.Method()),
						"path", utils.B2S(ctx.Path()),
						"ip", ip,
						"origin", utils.B2S(ctx.Request.Header.Peek("Origin")),
					)
					ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
				} else {
					mills := time.Since(start) / time.Millisecond
					status := ctx.Response.StatusCode()

					logFn := zap.S().Debugw
					if mills >= 500 {
						logFn = zap.S().Infow
					}
					if status >= 500 {
						logFn = zap.S().Errorw
					}

					logFn("gql request",
						"status", status,
						"duration", int(mills),
						"method", utils.B2S(ctx.Method()),
						"path", utils.B2S(ctx.Path()),
						"ip", ip,
						"origin", utils.B2S(ctx.Request.Header.Peek("Origin")),
					)
				}
			}()

			ctx.Response.Header.Set("X-Node-Name", gctx.Config().K8S.NodeName)
			ctx.Response.Header.Set("X-Pod-Name", gctx.Config().K8S.PodName)

			if err := doCORS(ctx); err != nil {
				return
			}

			if ctx.IsOptions() {
				ctx.SetStatusCode(fasthttp.StatusNoContent)
				return
			}

			if err := doAuth(ctx); err != nil {
				ctx.Response.Header.Add("X-Auth-Failure", err.Message())
			}

			router.Handler(ctx)
		},
		ReadTimeout:        time.Second * 10,
		WriteTimeout:       time.Second * 10,
		CloseOnShutdown:    true,
		Name:               "7TV - GQL",
		ReadBufferSize:     int(1000 * 1024),     // 32KB
		MaxRequestBodySize: int(6 * 1024 * 1024), // 6MB
	}

	go func() {
		<-gctx.Done()

		_ = server.Shutdown()
	}()

	return server.ListenAndServe(fmt.Sprintf("%s:%d", gctx.Config().Http.Addr, port))
}
