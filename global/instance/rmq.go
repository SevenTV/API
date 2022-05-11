package instance

import "github.com/streadway/amqp"

type Rmq interface {
	Subscribe(queue string) (<-chan amqp.Delivery, error)
	Publish(queue string, contentType string, deliveryMode uint8, msg []byte) error
}
