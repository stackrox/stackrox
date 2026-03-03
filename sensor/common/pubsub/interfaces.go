package pubsub

import "github.com/stackrox/rox/pkg/concurrency"

//go:generate mockgen-wrapper
type Event interface {
	Topic() Topic
	Lane() LaneID
}

type EventCallback func(Event) error

type LaneOption[T Lane] func(T)

type ConsumerOption[T Consumer] func(T)

type LaneConfig interface {
	NewLane() Lane
	LaneID() LaneID
}

type Lane interface {
	Publish(Event) error
	RegisterConsumer(ConsumerID, Topic, EventCallback) error
	Stop()
}

type NewConsumer func(laneID LaneID, topic Topic, consumerID ConsumerID, callback EventCallback) (Consumer, error)

type Consumer interface {
	Consume(concurrency.Waitable, Event) <-chan error
	Stop()
}
