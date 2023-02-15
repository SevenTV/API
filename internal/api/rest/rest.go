package rest

import (
	"fmt"
	"net"
	"time"

	"github.com/fasthttp/router"
	"github.com/seventv/api/internal/api/rest/portal"
	"github.com/seventv/api/internal/constant"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/middleware"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type HttpServer struct {
	listener net.Listener
	router   *router.Router
}

func New(gctx global.Context) error {
	var err error

	port := gctx.Config().Http.Ports.REST
	if port == 0 {
		port = 80
	}

	s := HttpServer{}

	s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", gctx.Config().Http.Addr, port))
	if err != nil {
		return err
	}

	s.router = router.New()

	// Add versions
	s.SetupHandlers()
	s.V3(gctx)
	s.V2(gctx)

	doAuth := middleware.Auth(gctx)
	doCORS := middleware.CORS(gctx)

	srv := &fasthttp.Server{
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
					zap.S().Errorw("panic in rest request handler",
						"panic", err,
						"status", ctx.Response.StatusCode(),
						"duration", int(time.Since(start)/time.Millisecond),
						"method", utils.B2S(ctx.Method()),
						"path", utils.B2S(ctx.Path()),
						"ip", utils.B2S(ctx.Request.Header.Peek("Cf-Connecting-IP")),
						"origin", utils.B2S(ctx.Request.Header.Peek("Origin")),
					)
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

					logFn("rest request",
						"status", status,
						"duration", int(mills),
						"method", utils.B2S(ctx.Method()),
						"path", utils.B2S(ctx.Path()),
						"ip", utils.B2S(ctx.Request.Header.Peek("Cf-Connecting-IP")),
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

			// Routing
			ctx.Response.Header.Set("Content-Type", "application/json") // default to JSON

			if err := doAuth(ctx); err != nil {
				ctx.Response.Header.Add("X-Auth-Failure", err.Message())
			}

			s.router.Handler(ctx)
		},
		ReadTimeout:                  time.Second * 600,
		IdleTimeout:                  time.Second * 10,
		ReadBufferSize:               int(32 * 1024),       // 32KB
		MaxRequestBodySize:           int(6 * 1024 * 1024), // 6MB
		DisablePreParseMultipartForm: true,
		StreamRequestBody:            true,
		CloseOnShutdown:              true,
	}

	// Dev Portal
	go func() {
		portal.Serve(gctx)
	}()

	// Gracefully exit when the global context is canceled
	go func() {
		<-gctx.Done()

		_ = srv.Shutdown()
	}()

	return srv.Serve(s.listener)
}
