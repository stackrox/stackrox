package connection

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Manager is responsible for managing all active connections from sensors.
//
//go:generate mockgen-wrapper
type Manager interface {
	// Need to register cluster manager to avoid cyclic dependencies with cluster datastore
	Start(mgr common.ClusterManager,
		netEntitiesMgr common.NetworkEntityManager,
		policyMgr common.PolicyManager,
		baselineMgr common.ProcessBaselineManager,
		networkBaselineMgr common.NetworkBaselineManager,
		delegatedRegistryConfigMgr common.DelegatedRegistryConfigManager,
		imageIntegrationMgr common.ImageIntegrationManager,
		complianceOperatorMgr common.ComplianceOperatorManager,
		autoTriggerUpgrades *concurrency.Flag) error

	// Connection-related methods.
	HandleConnection(ctx context.Context, sensorHello *central.SensorHello, cluster *storage.Cluster, eventPipeline pipeline.ClusterPipeline, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection
	CloseConnection(clusterID string)
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
