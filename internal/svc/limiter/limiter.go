package limiter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/seventv/api/internal/api/gql/v3/auth"
	"github.com/seventv/api/internal/constant"
	"github.com/seventv/common/redis"
	"github.com/seventv/common/utils"
	"go.uber.org/zap"
)

type Instance interface {
	AwaitMutation(ctx context.Context) func()
	Test(ctx context.Context, bucket string, limit int64, dur time.Duration, opt TestOptions) bool

	ScriptOk(ctx context.Context) bool
	LoadScript(ctx context.Context) error
	GetScript() string
}

type limiterInst struct {
	redis  redis.Instance
	script string

	mx *sync.Mutex
}

func New(ctx context.Context, rdis redis.Instance) (Instance, error) {
	l := limiterInst{
		redis: rdis,
		mx:    &sync.Mutex{},
	}

	if err := l.LoadScript(ctx); err != nil {
		return &l, err
	}

	return &l, nil
}

func (inst *limiterInst) ScriptOk(ctx context.Context) bool {
	ok, _ := inst.redis.RawClient().ScriptExists(ctx, inst.script).Result()
	if len(ok) == 0 || !ok[0] {
		return false
	}

	return true
}

func (inst *limiterInst) GetScript() string {
	return inst.script
}

func (inst *limiterInst) LoadScript(ctx context.Context) error {
	inst.mx.Lock()
	defer inst.mx.Unlock()

	var err error

	inst.script, err = inst.redis.RawClient().ScriptLoad(ctx, `
		local key = ARGV[1]
		local expire = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local by = tonumber(ARGV[4])
		
		local exists = redis.call("EXISTS", key)
		
		local count = redis.call("INCRBY", key, by)
		
		if exists == 0 then
			redis.call("EXPIRE", key, expire)
			return {count, expire}
		end
		
		local ttl = redis.call("TTL", key)
		
		return {count, ttl}
		
`).Result()
	if err != nil {
		return err
	}

	return nil
}

func (inst *limiterInst) AwaitMutation(ctx context.Context) func() {
	actor := auth.For(ctx)
	if actor.ID.IsZero() {
		return func() {}
	}

	k := inst.redis.ComposeKey("api-global", "rl", actor.ID.Hex(), "mutation_lock")

	mx := inst.redis.Mutex(k, time.Second*20)

	if err := mx.Lock(); err != nil {
		zap.S().Errorw("limiter, failed to acquire mutex", "key", k, "error", err)
	}

	return func() {
		if _, err := mx.Unlock(); err != nil {
			zap.S().Errorw("limiter, failed to release mutex", "key", k, "error", err)
		}
	}
}

func (inst *limiterInst) Test(ctx context.Context, bucket string, limit int64, dur time.Duration, opt TestOptions) bool {
	identifier := ""

	actor := auth.For(ctx)
	if !actor.ID.IsZero() {
		identifier = actor.ID.Hex()
	} else {
		ip := ctx.Value(constant.ClientIP)

		switch v := ip.(type) {
		case string:
			identifier = v
		default:
			identifier = "any"
		}
	}

	h := sha256.New()
	h.Write(utils.S2B(identifier))
	h.Write(utils.S2B(bucket))

	k := inst.redis.ComposeKey("api-global", "rl", hex.EncodeToString(h.Sum(nil)))

	rem := limit

	if res, err := inst.redis.RawClient().EvalSha(
		ctx,
		inst.GetScript(),
		[]string{},
		k.String(),
		dur.Seconds(),
		limit,
		utils.Ternary(opt.Incr > 0, opt.Incr, 1),
	).Result(); err != nil {
		zap.S().Errorw("limiter, failed to test", "key", k, "error", err)

		return true
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

		rem -= a[0]
	}

	if rem <= 0 {
		return false
	}

	return true
}

type TestOptions struct {
	Incr uint32
}
