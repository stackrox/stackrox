package events

import (
	"github.com/stackrox/rox/pkg/concurrency"
)

// Stream is an interface for the administration events stream.
type Stream interface {
	Consume(waitable concurrency.Waitable) *AdministrationEvent
	Produce(event *AdministrationEvent)
}
