package events

import (
	"net/http"

	"github.com/stackrox/rox/generated/storage"
)

// Handler is an interface to handle storage.Events.
type Handler interface {
	AddEventAsync(event *storage.Event)
	Stop()
	SSEHandler() http.HandlerFunc

	// TODO(dhaus): Would require to change the interface exposed, i.e. within pkg scope Stop/SSEHandler do not matter,
	// this is a central only implementation and should be moved there theoretically. Skipping for now.
}
