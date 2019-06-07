package connection

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
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

func (m *manager) HandleConnection(ctx context.Context, clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer,
	clusterMgr ClusterManager) error {
	conn, err := newConnection(ctx, clusterID, pf, clusterMgr)
	if err != nil {
		return errors.Wrap(err, "creating sensor connection")
	}

	var oldConnection *sensorConnection
	concurrency.WithLock(&m.connectionsByClusterIDMutex, func() {
		oldConnection = m.connectionsByClusterID[clusterID]
		m.connectionsByClusterID[clusterID] = conn
	})

	if oldConnection != nil {
		oldConnection.Terminate(errors.New("replaced by new connection"))
	}

	err = conn.Run(ctx, server)
	log.Warnf("Connection to server in cluster %s terminated: %v", clusterID, err)

	concurrency.WithLock(&m.connectionsByClusterIDMutex, func() {
		currentConn := m.connectionsByClusterID[clusterID]
		if currentConn == conn {
			delete(m.connectionsByClusterID, clusterID)
		}
	})

	return err
}

func (m *manager) GetActiveConnections() []SensorConnection {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	result := make([]SensorConnection, 0, len(m.connectionsByClusterID))

	for _, conn := range m.connectionsByClusterID {
		result = append(result, conn)
	}

	return result
}
