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
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
}

// Manager is responsible for managing all active connections from sensors.
//go:generate mockgen-wrapper
type Manager interface {
	// Need to register cluster manager to avoid cyclic dependencies with cluster datastore
	Start(mgr ClusterManager) error

	// Connection-related methods.
	HandleConnection(ctx context.Context, clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection
	GetActiveConnections() []SensorConnection

	// Upgrade-related methods.
	RecordUpgradeProgress(clusterID, upgradeProcessID string, upgradeProgress *storage.UpgradeProgress) error
	TriggerUpgrade(ctx context.Context, clusterID string) error
}
