package streamer

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Stream is an abstraction for a stream over which to receive sensor events and send out enforcement actions.
type Stream interface {
	Send(enforcement *v1.SensorEnforcement) error
	Recv() (*v1.SensorEvent, error)
}
