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

			lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			mqDown = gctx.Inst().MessageQueue != nil && !gctx.Inst().MessageQueue.Connected(lCtx)
			cancel()
			if mqDown {
				zap.S().Warnw("mq is not connected")
			}

			if gctx.Config().S3.Enabled && gctx.Inst().S3 != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if _, err := gctx.Inst().S3.ListBuckets(lCtx); err != nil {
					s3Down = true
					zap.S().Warnw("s3 is not responding",
						"error", err,
					)
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
