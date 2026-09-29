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
	// Consume processes event and reports the outcome on the returned channel.
	// Implementations must not rely on that channel being read: concurrentLane
	// dispatches Consume() without waiting on or draining the result, so a
	// consumer that only reports errors through this channel will have them
	// silently dropped there. Implementations should report their own errors
	// (e.g. logging/metrics) rather than depending solely on the caller reading
	// this channel.
	Consume(concurrency.Waitable, Event) <-chan error
	Stop()
}
