package global

import (
	"context"
	"time"

	"github.com/seventv/api/global/configure"
)

type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Err() error
	Value(key interface{}) interface{}
	Done() <-chan struct{}
	Config() *configure.Config
	Inst() *Instances
}

type gCtx struct {
	ctx    context.Context
	config *configure.Config
	inst   *Instances
}

func (g *gCtx) Deadline() (time.Time, bool) {
	return g.ctx.Deadline()
}

func (g *gCtx) Done() <-chan struct{} {
	return g.ctx.Done()
}

func (g *gCtx) Err() error {
	return g.ctx.Err()
}

func (g *gCtx) Value(key interface{}) interface{} {
	return g.ctx.Value(key)
}

func (g *gCtx) Config() *configure.Config {
	return g.config
}

func (g *gCtx) Inst() *Instances {
	return g.inst
}

func New(ctx context.Context, config *configure.Config) Context {
	return &gCtx{
		ctx:    ctx,
		config: config,
		inst:   &Instances{},
	}
}

func WithCancel(ctx Context) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithCancel(ctx)

	return &gCtx{
		ctx:    c,
		config: cfg,
		inst:   inst,
	}, cancel
}

func WithDeadline(ctx Context, deadline time.Time) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithDeadline(ctx, deadline)

	return &gCtx{
		ctx:    c,
		config: cfg,
		inst:   inst,
	}, cancel
}

func WithValue(ctx Context, key interface{}, value interface{}) Context {
	cfg := ctx.Config()
	inst := ctx.Inst()

	return &gCtx{
		ctx:    context.WithValue(ctx, key, value),
		config: cfg,
		inst:   inst,
	}
}

func WithTimeout(ctx Context, timeout time.Duration) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithTimeout(ctx, timeout)

	return &gCtx{
		ctx:    c,
		config: cfg,
		inst:   inst,
	}, cancel
}
