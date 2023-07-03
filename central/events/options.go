package events

import (
	"github.com/stackrox/rox/central/events/handler"
	"github.com/stackrox/rox/pkg/logging"
)

// WriteToEventStream provides a convenient wrapper to include log entries also within the event stream to make
// them discoverable to users.
func WriteToEventStream() logging.Options {
	return logging.Options{
		Output:       logging.StdoutAndEventStream,
		EventHandler: handler.Singleton(),
	}
}
