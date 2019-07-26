package connection

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ClusterManager envelopes functions that interact with clusters
type ClusterManager interface {
	UpdateClusterContactTimes(ctx context.Context, time time.Time, clusterID ...string) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
}

// Manager is responsible for managing all active connections from sensors.
//go:generate mockgen-wrapper
type Manager interface {
	Start()
	RegisterClusterManager(mgr ClusterManager)
	HandleConnection(ctx context.Context, clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection

	GetActiveConnections() []SensorConnection
}
