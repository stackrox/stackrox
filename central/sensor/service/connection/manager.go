package connection

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// ClusterManager envelopes functions that interact with clusters
type ClusterManager interface {
	UpdateClusterContactTimes(ctx context.Context, time time.Time, clusterID ...string) error
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
}

// PolicyManager implements an interface to retrieve policies
type PolicyManager interface {
	GetAllPolicies(ctx context.Context) ([]*storage.Policy, error)
}

// WhitelistManager implements an interface to retrieve whitelists
type WhitelistManager interface {
	WalkAll(ctx context.Context, fn func(whitelist *storage.ProcessWhitelist) error) error
}

// Manager is responsible for managing all active connections from sensors.
//go:generate mockgen-wrapper
type Manager interface {
	// Need to register cluster manager to avoid cyclic dependencies with cluster datastore
	Start(mgr ClusterManager, policyMgr PolicyManager, whitelistMgr WhitelistManager, autoTriggerUpgrades *concurrency.Flag) error

	// Connection-related methods.
	HandleConnection(ctx context.Context, clusterID string, eventPipeline pipeline.ClusterPipeline, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection
	GetActiveConnections() []SensorConnection
	BroadcastMessage(msg *central.MsgToSensor)
	SendMessage(clusterID string, msg *central.MsgToSensor) error

	// Upgrade-related methods.
	TriggerUpgrade(ctx context.Context, clusterID string) error
	ProcessCheckInFromUpgrader(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error)
	ProcessUpgradeCheckInFromSensor(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromSensorRequest) error
}
