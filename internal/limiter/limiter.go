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
}

type limiterInst struct {
	redis redis.Instance
	// script string

	mx *sync.Mutex
}

func New(ctx context.Context, rdis redis.Instance) (Instance, error) {
	l := limiterInst{
		redis: rdis,
		mx:    &sync.Mutex{},
	}

	/*
		if err := l.LoadScripts(ctx); err != nil {
			return &l, err
		}
	*/

	return &l, nil
}

/*
func (inst *limiterInst) ScriptOk(ctx context.Context) bool {
	ok, _ := inst.redis.RawClient().ScriptExists(ctx, inst.script).Result()
	if len(ok) == 0 || !ok[0] {
		return false
	}

	return true
}

func (inst *limiterInst) LoadScripts(ctx context.Context) error {
	inst.mx.Lock()
	defer inst.mx.Unlock()

	script, err := os.ReadFile("./internal/limiter/limiter.lua")
	if err != nil {
		return err
	}

	inst.script, err = inst.redis.RawClient().ScriptLoad(ctx, utils.B2S(script)).Result()
	if err != nil {
		return err
	}

	return nil
}
*/

func (inst *limiterInst) AwaitMutation(ctx context.Context) func() {
	actor := auth.For(ctx)
	if actor == nil {
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
