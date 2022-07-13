package global

import (
	"context"
	"time"

	"github.com/seventv/api/internal/configure"
	"github.com/seventv/api/internal/instance"
)

type Context interface {
	context.Context
	Config() *configure.Config
	Inst() *instance.Instances
}

type gCtx struct {
	context.Context
	config *configure.Config
	inst   *instance.Instances
}

func (g *gCtx) Config() *configure.Config {
	return g.config
}

func (g *gCtx) Inst() *instance.Instances {
	return g.inst
}

func New(ctx context.Context, config *configure.Config) Context {
	return &gCtx{
		Context: ctx,
		config:  config,
		inst:    &instance.Instances{},
	}
}

func WithCancel(ctx Context) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithCancel(ctx)

	return &gCtx{
		Context: c,
		config:  cfg,
		inst:    inst,
	}, cancel
}

func WithDeadline(ctx Context, deadline time.Time) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithDeadline(ctx, deadline)

	return &gCtx{
		Context: c,
		config:  cfg,
		inst:    inst,
	}, cancel
}

func WithValue(ctx Context, key interface{}, value interface{}) Context {
	cfg := ctx.Config()
	inst := ctx.Inst()

	return &gCtx{
		Context: context.WithValue(ctx, key, value),
		config:  cfg,
		inst:    inst,
	}
}

func WithTimeout(ctx Context, timeout time.Duration) (Context, context.CancelFunc) {
	cfg := ctx.Config()
	inst := ctx.Inst()

	c, cancel := context.WithTimeout(ctx, timeout)

	return &gCtx{
		Context: c,
		config:  cfg,
		inst:    inst,
	}, cancel
}
