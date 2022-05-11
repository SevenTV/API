package rmq

import (
	"context"
	"time"

	"github.com/seventv/api/internal/instance"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type Options struct {
	URI string
}

type Instance struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	opts Options
}

func New(ctx context.Context, opts Options) (instance.RMQ, error) {
	i := &Instance{
		opts: opts,
	}

	if err := i.connect(); err != nil {
		return nil, err
	}

	go i.keepalive(ctx)

	return i, nil
}

func (i *Instance) connect() error {
	if i.conn != nil {
		_ = i.conn.Close()
	}

	conn, err := amqp.Dial(i.opts.URI)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	i.conn = conn
	i.ch = ch

	return nil
}

func (i *Instance) keepalive(ctx context.Context) {
	timer := time.NewTimer(time.Millisecond * 500)
	attempts := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if i.conn.IsClosed() {
				if err := i.connect(); err != nil {
					attempts++
					if attempts > 10 {
						zap.S().Fatalw("failed to connect to rmq",
							"error", err,
							"attempt", attempts,
						)
					}

					zap.S().Errorw("failed to connect to rmq",
						"error", err,
						"attempt", attempts,
					)
					timer.Reset(time.Second * 5)
				} else if attempts != 0 {
					attempts = 0
					timer.Reset(time.Millisecond * 500)
				}
			}
		}
	}
}

func (i *Instance) Connected() bool {
	if i.conn == nil {
		return false
	}

	return !i.conn.IsClosed()
}

func (i *Instance) Publish(req instance.RmqPublishRequest) error {
	return i.ch.Publish(req.Exchange, req.RoutingKey, req.Mandatory, req.Immediate, req.Publishing)
}

func (i *Instance) Subscribe(ctx context.Context, req instance.RmqSubscribeRequest) (<-chan amqp.Delivery, error) {
	c, err := i.conn.Channel()
	if err != nil {
		return nil, err
	}

	ch, err := c.Consume(
		req.Queue,
		req.Consumer,
		req.AutoAck,
		req.Exclusive,
		req.NoLocal,
		req.NoWait,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	go func() {
		defer c.Close()
		<-ctx.Done()
	}()

	return ch, err
}
