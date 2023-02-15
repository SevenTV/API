package middleware

import (
	"strconv"
	"time"

	"github.com/seventv/api/internal/api/rest/rest"
	"github.com/seventv/api/internal/constant"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/middleware"
	"github.com/seventv/common/errors"
	"go.uber.org/zap"
)

func RateLimit(gctx global.Context, bucket string, rate [2]int64) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		identifier, _ := ctx.UserValue(constant.ClientIP).String()

		actor, ok := ctx.GetActor()
		if ok {
			identifier = actor.ID.Hex()
		}

		if identifier == "" {
			return nil
		}

		limit, remaining, ttl, err := middleware.DoRateLimit(gctx, ctx, bucket, rate[0], identifier, time.Second*time.Duration(rate[1]))
		if err != nil {
			switch e := err.(type) {
			case errors.APIError:
				return e
			}

			zap.S().Errorw("Error while rate limiting a request", "error", err)
		}

		// Apply headers
		ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(int(limit)))
		ctx.Response.Header.Set("X-RateLimit-Remaining", strconv.Itoa(int(remaining)))
		ctx.Response.Header.Set("X-RateLimit-Reset", strconv.Itoa(int(ttl)))

		if remaining < 1 {
			return errors.ErrRateLimited()
		}

		return nil
	}
}
