package health

import (
	"context"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func New(gctx global.Context) <-chan struct{} {
	done := make(chan struct{})

	srv := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			defer func() {
				if err := recover(); err != nil {
					zap.S().Errorw("panic in health",
						"panic", err,
					)
				}
			}()

			var (
				mqDown    bool
				s3Down    bool
				redisDown bool
				mongoDown bool
			)

			if gctx.Inst().Redis != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if err := gctx.Inst().Redis.Ping(lCtx); err != nil {
					zap.S().Warnw("redis is not responding",
						"error", err,
					)
					redisDown = true
				}
				cancel()
			}

			if gctx.Inst().Mongo != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if err := gctx.Inst().Mongo.Ping(lCtx); err != nil {
					mongoDown = true
					zap.S().Warnw("mongo is not responding",
						"error", err,
					)
				}
				cancel()
			}

			if mqDown || s3Down || redisDown || mongoDown {
				ctx.SetStatusCode(500)
			}
		},
	}

	go func() {
		defer close(done)
		zap.S().Infow("Health enabled",
			"bind", gctx.Config().Health.Bind,
		)

		if err := srv.ListenAndServe(gctx.Config().Health.Bind); err != nil {
			zap.S().Fatalw("failed to bind health",
				"error", err,
			)
		}
	}()

	go func() {
		<-gctx.Done()

		_ = srv.Shutdown()
	}()

	return done
}
