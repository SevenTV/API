package rmq

import (
	"context"
	"sync"
	"time"

	"github.com/seventv/api/internal/instance"
	"github.com/seventv/common/sync_map"
	"github.com/streadway/amqp"
)

type mockState struct {
	arr []amqp.Delivery
	mtx sync.Mutex
}

type mockAcknowledger struct{}

func (mockAcknowledger) Ack(tag uint64, multiple bool) error {
	return nil
}

func (mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	return nil
}

func (mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

type MockInstance struct {
	mp        *sync_map.Map[string, *mockState]
	connected bool
	mtx       sync.Mutex
}

func NewMock() (instance.RMQ, error) {
	return &MockInstance{
		mp:        &sync_map.Map[string, *mockState]{},
		connected: true,
	}, nil
}

func (i *MockInstance) Connected() bool {
	return i.connected
}

func (i *MockInstance) SetConnected(connected bool) {
	i.mtx.Lock()
	i.connected = connected
	i.mtx.Unlock()
}

func (i *MockInstance) Publish(req instance.RmqPublishRequest) error {
	i.mtx.Lock()
	if !i.connected {
		i.mtx.Unlock()
		return amqp.ErrClosed
	}
	i.mtx.Unlock()

	v, _ := i.mp.LoadOrStore(req.RoutingKey, &mockState{})
	v.mtx.Lock()
	defer v.mtx.Unlock()

	body := make([]byte, len(req.Publishing.Body))
	copy(body, req.Publishing.Body)

	v.arr = append(v.arr, amqp.Delivery{
		Headers:         req.Publishing.Headers,
		Acknowledger:    mockAcknowledger{},
		ContentType:     req.Publishing.ContentType,
		ContentEncoding: req.Publishing.ContentEncoding,
		DeliveryMode:    req.Publishing.DeliveryMode,
		Priority:        req.Publishing.Priority,
		CorrelationId:   req.Publishing.CorrelationId,
		ReplyTo:         req.Publishing.ReplyTo,
		Expiration:      req.Publishing.Expiration,
		MessageId:       req.Publishing.MessageId,
		Timestamp:       req.Publishing.Timestamp,
		Type:            req.Publishing.Type,
		UserId:          req.Publishing.UserId,
		AppId:           req.Publishing.AppId,
		Body:            body,
		Exchange:        req.Exchange,
		RoutingKey:      req.RoutingKey,
	})
	return nil
}

func (i *MockInstance) Subscribe(ctx context.Context, req instance.RmqSubscribeRequest) (<-chan amqp.Delivery, error) {
	v, _ := i.mp.LoadOrStore(req.Queue, &mockState{})

	ch := make(chan amqp.Delivery)
	go func() {
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				i.mtx.Lock()
				if !i.connected {
					i.mtx.Unlock()
					return
				}
				i.mtx.Unlock()

				v.mtx.Lock()
				for len(v.arr) != 0 {
					msg := v.arr[len(v.arr)-1]
					v.arr = v.arr[:len(v.arr)-1]
					ch <- msg
				}
				v.mtx.Unlock()
			}
		}
	}()
	return ch, nil
}
