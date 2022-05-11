package rmq

import (
	"context"

	"github.com/seventv/api/internal/instance"
	"github.com/streadway/amqp"
)

type Options struct {
	URI string
}

type Instance struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(ctx context.Context, o Options) (instance.RMQ, error) {
	conn, err := amqp.Dial(o.URI)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Instance{
		ch:   ch,
		conn: conn,
	}, nil
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
