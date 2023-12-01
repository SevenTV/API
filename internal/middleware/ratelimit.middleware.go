package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/seventv/api/internal/constant"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func RateLimit(gctx global.Context, bucket string, limit int64, ex time.Duration) Middleware {
	return func(ctx *fasthttp.RequestCtx) errors.APIError {
		var identifier string
		switch t := ctx.UserValue(constant.ClientIP).(type) {
		case string:
			identifier = t
		}

		var actor *structures.User
		switch t := ctx.Value("user").(type) {
		case *structures.User:
			actor = t
		}

		if actor != nil {
			identifier = actor.ID.Hex()
		}

		if identifier == "" {
			return nil
		}

		limit, remaining, ttl, err := DoRateLimit(gctx, ctx, bucket, limit, identifier, ex)
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

func DoRateLimit(
	gctx global.Context,
	ctx context.Context,
	bucket string,
	limit int64,
	identifier string,
	ex time.Duration,
) (int64, int64, int64, error) {
	h := sha256.New()
	h.Write(utils.S2B(identifier))
	h.Write(utils.S2B(bucket))

	// State
	remaining := int64(limit)
	reset := int64(0)

	// Check script
	var err error
	if ok := gctx.Inst().Limiter.ScriptOk(ctx); !ok {
		err = gctx.Inst().Limiter.LoadScript(ctx)
	}

	if err != nil {
		zap.S().Errorw("Error loading redis luascript for rate limiting", "error", err)
		return 0, 0, 0, errors.ErrInternalServerError()
	}

	k := gctx.Inst().Redis.ComposeKey("api", "rl", bucket, hex.EncodeToString(h.Sum(nil)))

	if res, err := gctx.Inst().Redis.RawClient().EvalSha(
		ctx,
		gctx.Inst().Limiter.GetScript(),
		[]string{},
		k.String(),
		ex.Seconds(),
		limit,
		1,
	).Result(); err != nil {
		return 0, 0, 0, errors.ErrInternalServerError().SetDetail("Rate Limiter Error:", err)
	} else {
		a := make([]int64, 3)

		result := []any{}
		switch t := res.(type) {
		case []any:
			result = t
		}

		for i, v := range result {
			var val int64
			switch t := v.(type) {
			case int64:
				val = t
			}
			a[i] = val
		}

		remaining -= a[0]
		reset = a[1]
	}

	return limit, remaining, reset, nil
}
