package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Stream is an abstraction for a stream over which to receive sensor events and send out enforcement actions.
type Stream interface {
	Send(enforcement *central.SensorEnforcement) error
	Recv() (*central.SensorEvent, error)
}
