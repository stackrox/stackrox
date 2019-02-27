package streamer

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

type managerImpl struct {
	streamers      map[string]Streamer
	streamersMutex sync.Mutex
}

func (m *managerImpl) CreateStreamer(clusterID string, pf pipeline.Factory) (Streamer, error) {
	m.streamersMutex.Lock()
	defer m.streamersMutex.Unlock()

	if conn := m.streamers[clusterID]; conn != nil {
		return nil, fmt.Errorf("there already is an active connection for cluster %s", clusterID)
	}

	pl, err := pf.PipelineForCluster(clusterID)
	if err != nil {
		return nil, err
	}

	streamer := NewStreamer(clusterID, pl)
	if err != nil {
		return nil, err
	}
	m.streamers[clusterID] = streamer
	return streamer, nil
}

func (m *managerImpl) GetStreamer(clusterID string) Streamer {
	m.streamersMutex.Lock()
	defer m.streamersMutex.Unlock()

	return m.streamers[clusterID]
}

func (m *managerImpl) RemoveStreamer(clusterID string, streamer Streamer) error {
	m.streamersMutex.Lock()
	defer m.streamersMutex.Unlock()

	existingStreamer := m.streamers[clusterID]
	if existingStreamer == streamer {
		delete(m.streamers, clusterID)
		return nil
	}

	if existingStreamer == nil {
		return fmt.Errorf("no active sensor streamer for cluster %s", clusterID)
	}
	return fmt.Errorf("sensor streamer to be removed is not the active connection for cluster %s", clusterID)
}
