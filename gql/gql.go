package gql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/utils"
	"github.com/fasthttp/router"
	"github.com/seventv/api/global"

	"github.com/seventv/api/global/middleware"
	v2 "github.com/seventv/api/gql/v2"
	v3 "github.com/seventv/api/gql/v3"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})

	gqlv3 := v3.GqlHandlerV3(gCtx)
	gqlv2 := v2.GqlHandlerV2(gCtx)

	router := router.New()

	router.RedirectTrailingSlash = true
	mid := func(ctx *fasthttp.RequestCtx) {
		if err := middleware.Auth(gCtx)(ctx); err != nil {
			ctx.Response.Header.Add("X-Auth-Failure", err.Message())
			goto handler
		}

	handler:
		switch ctx.UserValue("v") {
		case "v3":
			gqlv3(ctx)
		case "v2":
			gqlv2(ctx)
		default:
			err := errors.ErrUnknownRoute()
			b, _ := json.Marshal(map[string]interface{}{
				"error":      err.Message(),
				"error_code": err.Code(),
			})
			_, _ = ctx.Write(b)
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}
	router.GET("/{v}", mid)
	router.POST("/{v}", mid)

	router.HandleOPTIONS = true
	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				var err interface{}
				if err != nil {
					ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
				}
				l := logrus.WithFields(logrus.Fields{
					"status":     ctx.Response.StatusCode(),
					"duration":   int64(time.Since(start) / time.Millisecond),
					"entrypoint": "api",
					"path":       utils.B2S(ctx.Path()),
					"ip":         utils.B2S(ctx.Response.Header.Peek("Cf-Connecting-IP")),
					"origin":     utils.B2S(ctx.Response.Header.Peek("Origin")),
				})
				if err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info("")
				}
			}()
			// CORS
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
			ctx.Response.Header.Set("Access-Control-Expose-Headers", "X-Collection-Size")
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "*")
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			if ctx.IsOptions() {
				ctx.SetStatusCode(fasthttp.StatusNoContent)
				return
			}

			router.Handler(ctx)
		},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
		CloseOnShutdown: true,
		Name:            "7TV - GQL",
	}

	go func() {
		port := gCtx.Config().Http.Ports.GQL
		if port == 0 {
			port = 80
		}
		if err := server.ListenAndServe(fmt.Sprintf("%s:%d", gCtx.Config().Http.Addr, port)); err != nil {
			logrus.Fatal("failed to start api server: ", err)
		}
		close(done)
	}()

	go func() {
		<-gCtx.Done()
		_ = server.Shutdown()
	}()

	return done
}
