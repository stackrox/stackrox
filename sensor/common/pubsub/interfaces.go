package pubsub

import "github.com/stackrox/rox/pkg/concurrency"

//go:generate mockgen-wrapper
type Event interface {
	Topic() Topic
	Lane() LaneID
}

// Mergeable is an optional interface that events can implement to merge
// state from an existing event when a duplicate key is detected in a
// deduplicating lane. If the new event implements Mergeable, MergeFrom
// is called with the old event before the old event is removed from the queue.
type Mergeable[K comparable] interface {
	Event
	MergeFrom(old Event)
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
