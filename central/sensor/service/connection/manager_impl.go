package connection

import (
	"errors"
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

type manager struct {
	connectionsByClusterID      map[string]*sensorConnection
	connectionsByClusterIDMutex sync.RWMutex
}

func newManager() *manager {
	return &manager{
		connectionsByClusterID: make(map[string]*sensorConnection),
	}
}

func (m *manager) GetConnection(clusterID string) SensorConnection {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	conn := m.connectionsByClusterID[clusterID]
	if conn == nil {
		return nil
	}
	return conn
}

func (m *manager) HandleConnection(clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer) error {
	conn, err := newConnection(clusterID, pf)

	if err != nil {
		return fmt.Errorf("creating sensor connection: %v", err)
	}

	var oldConnection *sensorConnection
	concurrency.WithLock(&m.connectionsByClusterIDMutex, func() {
		oldConnection = m.connectionsByClusterID[clusterID]
		m.connectionsByClusterID[clusterID] = conn
	})

	if oldConnection != nil {
		oldConnection.Terminate(errors.New("replaced by new connection"))
	}

	return conn.Run(server)
}
