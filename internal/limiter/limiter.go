package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/seventv/api/internal/gql/v3/auth"
	"github.com/seventv/common/redis"
	"go.uber.org/zap"
)

type Instance interface {
	AwaitMutation(ctx context.Context) func()

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
		
		if count > limit then
			return {redis.call("DECRBY", key, by), ttl, 1}
		end
		
		return {count, ttl, 0}
		
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
