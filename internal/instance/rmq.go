package instance

import (
	"context"

	"github.com/streadway/amqp"
)

type RMQ interface {
	Publish(req RmqPublishRequest) error
	Subscribe(ctx context.Context, req RmqSubscribeRequest) (<-chan amqp.Delivery, error)
}

type RmqPublishRequest struct {
	Exchange   string
	RoutingKey string
	Mandatory  bool
	Immediate  bool
	Publishing amqp.Publishing
}

type RmqSubscribeRequest struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}
