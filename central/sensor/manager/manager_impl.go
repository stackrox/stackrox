package manager

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/sensorevent/service/streamer"
	"github.com/stackrox/rox/central/sensornetworkflow"
)

type sensorManager struct {
	connections      map[string]SensorConnection
	connectionsMutex sync.Mutex

	eventStreamManager streamer.Manager
	flowClusterStore   store.ClusterStore
}

// New creates and returns a new SensorManager.
func New(eventStreamManager streamer.Manager, flowClusterStore store.ClusterStore) SensorManager {
	return &sensorManager{
		connections:        make(map[string]SensorConnection),
		eventStreamManager: eventStreamManager,
		flowClusterStore:   flowClusterStore,
	}
}

func (m *sensorManager) CreateConnection(clusterID string) (SensorConnection, error) {
	m.connectionsMutex.Lock()
	defer m.connectionsMutex.Unlock()

	if conn := m.connections[clusterID]; conn != nil {
		return nil, fmt.Errorf("there already is an active connection for cluster %s", clusterID)
	}

	flowStore, err := m.flowClusterStore.CreateFlowStore(clusterID)
	if err != nil {
		return nil, fmt.Errorf("creating flow store: %v", err)
	}

	conn := newConnection()

	eventStreamer := m.eventStreamManager.CreateStreamer(clusterID)
	eventStreamer.Start(conn.newEventStream())
	go conn.runEventStreamer(eventStreamer)

	flowStream := conn.newNetworkFlowStream()
	flowHandler := sensornetworkflow.NewHandler(clusterID, flowStore, flowStream)
	go conn.runFlowHandler(flowHandler)

	m.connections[clusterID] = conn

	return conn, nil
}

func (m *sensorManager) RemoveConnection(clusterID string, connection SensorConnection) error {
	m.connectionsMutex.Lock()
	defer m.connectionsMutex.Unlock()

	existingConn := m.connections[clusterID]
	if existingConn == connection {
		delete(m.connections, clusterID)
		return nil
	}

	if existingConn == nil {
		return fmt.Errorf("no active sensor connection for cluster %s", clusterID)
	}
	return fmt.Errorf("sensor connection to be removed is not the active connection for cluster %s", clusterID)
}
