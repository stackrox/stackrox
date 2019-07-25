package connection

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	clusterCheckinInterval = 30 * time.Second
)

var (
	clusterCheckInContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
)

type manager struct {
	connectionsByClusterID      map[string]*sensorConnection
	connectionsByClusterIDMutex sync.RWMutex

	clusters ClusterManager
}

func newManager() *manager {
	return &manager{
		connectionsByClusterID: make(map[string]*sensorConnection),
	}
}

// Need to pass the ClusterManager because of cyclic dependencies
func (m *manager) Start() {
	t := time.NewTicker(clusterCheckinInterval)
	defer t.Stop()

	for range t.C {
		connections := m.GetActiveConnections()
		clusterIDs := make([]string, 0, len(connections))
		for _, c := range connections {
			clusterIDs = append(clusterIDs, c.ClusterID())
		}
		if err := m.clusters.UpdateClusterContactTimes(clusterCheckInContext, time.Now(), clusterIDs...); err != nil {
			log.Errorf("error checking in clusters: %v", err)
		}
	}
}

// Need to registry cluster manager to avoid cyclic dependencies with cluster datastore
func (m *manager) RegisterClusterManager(mgr ClusterManager) {
	m.clusters = mgr
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

func (m *manager) HandleConnection(ctx context.Context, clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer) error {
	conn, err := newConnection(ctx, clusterID, pf, m.clusters)
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
