package connection

import (
	"context"

	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// Manager is responsible for managing all active connections from sensors.
//go:generate mockgen-wrapper
type Manager interface {
	// Need to register cluster manager to avoid cyclic dependencies with cluster datastore
	Start(mgr common.ClusterManager,
		netEntitiesMgr common.NetworkEntityManager,
		policyMgr common.PolicyManager,
		baselineMgr common.ProcessBaselineManager,
		networkBaselineMgr common.NetworkBaselineManager,
		autoTriggerUpgrades *concurrency.Flag) error

	// Connection-related methods.
	HandleConnection(ctx context.Context, sensorHello *central.SensorHello, cluster *storage.Cluster, eventPipeline pipeline.ClusterPipeline, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection
	GetActiveConnections() []SensorConnection
	PreparePoliciesAndBroadcast(policies []*storage.Policy)
	BroadcastMessage(msg *central.MsgToSensor)
	SendMessage(clusterID string, msg *central.MsgToSensor) error

	// Upgrade-related methods.
	TriggerUpgrade(ctx context.Context, clusterID string) error
	TriggerCertRotation(ctx context.Context, clusterID string) error
	ProcessCheckInFromUpgrader(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error)
	ProcessUpgradeCheckInFromSensor(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromSensorRequest) error

	PushExternalNetworkEntitiesToSensor(ctx context.Context, clusterID string) error
	PushExternalNetworkEntitiesToAllSensors(ctx context.Context) error
}
