package streamer

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
)

type managerImpl struct {
	lock sync.Mutex

	pl pipeline.Pipeline

	clusterIDToStream map[string]Streamer
}

// GetStreamer fetches the Streamer for the given cluster, or nil if non exists.
func (s *managerImpl) GetStreamer(clusterID string) Streamer {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.clusterIDToStream[clusterID]
}

// CreateStreamer creates a Streamer for the given cluster. Returns an err if one already exists.
func (s *managerImpl) CreateStreamer(clusterID string) (Streamer, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Check that a stream does not already exist.
	if s.clusterIDToStream[clusterID] != nil {
		return nil, fmt.Errorf("stream for cluster ID %s is already open", clusterID)
	}

	pendingEvents := queue.NewEventQueue()

	st := NewStreamer(clusterID, pendingEvents, s.pl)
	s.clusterIDToStream[clusterID] = st
	return st, nil
}

// GetStreamer fetches the Streamer for the given cluster, or nil if non exists.
func (s *managerImpl) RemoveStreamer(clusterID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.clusterIDToStream[clusterID] == nil {
		return fmt.Errorf("stream for cluster ID %s is not open", clusterID)
	}

	delete(s.clusterIDToStream, clusterID)
	return nil
}
