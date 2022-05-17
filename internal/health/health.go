package health

import (
	"context"
	"time"

	"github.com/seventv/api/internal/global"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func New(gCtx global.Context) <-chan struct{} {
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
				rmqDown   bool
				s3Down    bool
				redisDown bool
				mongoDown bool
			)

			if gCtx.Inst().Redis != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if err := gCtx.Inst().Redis.Ping(lCtx); err != nil {
					zap.S().Warnw("redis is not responding",
						"error", err,
					)
					redisDown = true
				}
				cancel()
			}

			if gCtx.Inst().RMQ != nil && !gCtx.Inst().RMQ.Connected() {
				rmqDown = true
				zap.S().Warnw("rmq is not connected")
			}

			if gCtx.Inst().S3 != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if _, err := gCtx.Inst().S3.ListBuckets(lCtx); err != nil {
					s3Down = true
					zap.S().Warnw("s3 is not responding",
						"error", err,
					)
				}
				cancel()
			}

			if gCtx.Inst().Mongo != nil {
				lCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				if err := gCtx.Inst().Mongo.Ping(lCtx); err != nil {
					mongoDown = true
					zap.S().Warnw("mongo is not responding",
						"error", err,
					)
				}
				cancel()
			}

			if rmqDown || s3Down || redisDown || mongoDown {
				ctx.SetStatusCode(500)
			}
		},
	}

	go func() {
		defer close(done)
		zap.S().Infow("Health enabled",
			"bind", gCtx.Config().Health.Bind,
		)
		if err := srv.ListenAndServe(gCtx.Config().Health.Bind); err != nil {
			zap.S().Fatalw("failed to bind health",
				"error", err,
			)
		}
	}()

	go func() {
		<-gCtx.Done()
		_ = srv.Shutdown()
	}()

	return done
}
