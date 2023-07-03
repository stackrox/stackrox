package events

import "github.com/stackrox/rox/generated/storage"

// Handler is an interface to handle storage.Events.
type Handler interface {
	AddEventAsync(event *storage.Event)
	Stop()
}
