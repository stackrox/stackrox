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
	UpdateClusterContactTime(ctx context.Context, clusterID string, time time.Time) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
}

// Manager is responsible for managing all active connections from sensors.
//go:generate mockgen-wrapper Manager
type Manager interface {
	HandleConnection(ctx context.Context, clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer, clusterMgr ClusterManager) error
	GetConnection(clusterID string) SensorConnection

	GetActiveConnections() []SensorConnection
}
