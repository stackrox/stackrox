package events

import (
	"github.com/stackrox/rox/pkg/concurrency"
)

// Stream is an interface for the administration events stream.
type Stream interface {
	Consume(waitable concurrency.Waitable) func(yield func(*AdministrationEvent) bool)
	Produce(event *AdministrationEvent)
}
