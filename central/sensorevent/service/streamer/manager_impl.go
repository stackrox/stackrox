package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
)

type managerImpl struct {
	pl pipeline.Pipeline
}

// CreateStreamer creates a Streamer for the given cluster.
func (s *managerImpl) CreateStreamer(clusterID string) Streamer {
	pendingEvents := queue.NewEventQueue()

	st := NewStreamer(clusterID, pendingEvents, s.pl)
	return st
}
