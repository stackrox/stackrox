package connection

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clusterhealth"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	clusterCheckinInterval = 30 * time.Second

	connectionTerminationTimeout = 5 * time.Second
)

var (
	managerCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	clusterSAC = sac.ForResource(resources.Cluster)
)

func checkClusterWriteAccess(ctx context.Context, clusterID string) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

type connectionAndUpgradeController struct {
	connection  *sensorConnection
	upgradeCtrl upgradecontroller.UpgradeController
}

type manager struct {
	connectionsByClusterID      map[string]connectionAndUpgradeController
	connectionsByClusterIDMutex sync.RWMutex

	clusters                   common.ClusterManager
	networkEntities            common.NetworkEntityManager
	policies                   common.PolicyManager
	baselines                  common.ProcessBaselineManager
	networkBaselines           common.NetworkBaselineManager
	delegatedRegistryConfigMgr common.DelegatedRegistryConfigManager
	imageIntegrationMgr        common.ImageIntegrationManager
	manager                    hashManager.Manager
	complianceOperatorMgr      common.ComplianceOperatorManager
	rateLimitMgr               *rateLimitManager
	autoTriggerUpgrades        *concurrency.Flag
}

// NewManager returns a new connection manager
func NewManager(mgr hashManager.Manager) Manager {
	return &manager{
		connectionsByClusterID: make(map[string]connectionAndUpgradeController),
		manager:                mgr,
		rateLimitMgr:           newRateLimitManager(),
	}
}

func (m *manager) initializeUpgradeControllers() error {
	clusters, err := m.clusters.GetClusters(managerCtx)
	if err != nil {
		return err
	}

	m.connectionsByClusterIDMutex.Lock()
	defer m.connectionsByClusterIDMutex.Unlock()
	for _, cluster := range clusters {
		upgradeCtrl, err := upgradecontroller.New(cluster.GetId(), m.clusters, m.autoTriggerUpgrades)
		if err != nil {
			return err
		}
		m.connectionsByClusterID[cluster.GetId()] = connectionAndUpgradeController{
			upgradeCtrl: upgradeCtrl,
		}
	}
	return nil
}

func (m *manager) Start(clusterManager common.ClusterManager,
	networkEntityManager common.NetworkEntityManager,
	policyManager common.PolicyManager,
	baselineManager common.ProcessBaselineManager,
	networkBaselineManager common.NetworkBaselineManager,
	delegatedRegistryConfigManager common.DelegatedRegistryConfigManager,
	imageIntegrationMgr common.ImageIntegrationManager,
	complianceOperatorMgr common.ComplianceOperatorManager,
	autoTriggerUpgrades *concurrency.Flag,
) error {
	m.clusters = clusterManager
	m.networkEntities = networkEntityManager
	m.policies = policyManager
	m.baselines = baselineManager
	m.networkBaselines = networkBaselineManager
	m.delegatedRegistryConfigMgr = delegatedRegistryConfigManager
	m.imageIntegrationMgr = imageIntegrationMgr
	m.complianceOperatorMgr = complianceOperatorMgr
	m.autoTriggerUpgrades = autoTriggerUpgrades
	err := m.initializeUpgradeControllers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize upgrade controllers")
	}

	go m.updateClusterHealthForever()
	return nil
}

func (m *manager) updateClusterHealthForever() {
	t := time.NewTicker(clusterCheckinInterval)
	defer t.Stop()

	for range t.C {
		clusters, err := m.clusters.GetClusters(managerCtx)
		if err != nil {
			log.Errorf("error updating cluster healths: %v", err)
		}

		for _, cluster := range clusters {
			conn := m.GetConnection(cluster.GetId())
			if conn == nil {
				m.updateInactiveClusterHealth(cluster)
				continue
			}

			// Update cluster contact times for active sensors from here iff they do not have health monitoring capability.
			// Otherwise, rely on cluster health pipeline.
			if !conn.HasCapability(centralsensor.HealthMonitoringCap) {
				m.updateActiveClusterHealth(cluster)
			}
		}
	}
}

func (m *manager) updateInactiveClusterHealth(cluster *storage.Cluster) {
	oldHealth := cluster.GetHealthStatus()
	lastContact := protoconv.ConvertTimestampToTimeOrDefault(oldHealth.GetLastContact(), time.Time{})
	newSensorStatus := clusterhealth.PopulateInactiveSensorStatus(lastContact)
	clusterHealthStatus := &storage.ClusterHealthStatus{
		SensorHealthStatus:    newSensorStatus,
		CollectorHealthStatus: oldHealth.GetCollectorHealthStatus(),
		LastContact:           oldHealth.GetLastContact(),
		CollectorHealthInfo:   oldHealth.GetCollectorHealthInfo(),
		HealthInfoComplete:    oldHealth.GetHealthInfoComplete(),
	}
	clusterHealthStatus.OverallHealthStatus = clusterhealth.PopulateOverallClusterStatus(clusterHealthStatus)

	if err := m.clusters.UpdateClusterHealth(managerCtx, cluster.GetId(), clusterHealthStatus); err != nil {
		log.Errorf("error updating health for cluster %s (id: %s): %v", cluster.GetName(), cluster.GetId(), err)
	}
}

func (m *manager) updateActiveClusterHealth(cluster *storage.Cluster) {
	clusterHealthStatus := &storage.ClusterHealthStatus{
		SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
		CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
		LastContact:           types.TimestampNow(),
	}
	clusterHealthStatus.OverallHealthStatus = clusterhealth.PopulateOverallClusterStatus(clusterHealthStatus)

	if err := m.clusters.UpdateClusterHealth(managerCtx, cluster.GetId(), clusterHealthStatus); err != nil {
		log.Errorf("error updating health for cluster %s (id: %s): %v", cluster.GetName(), cluster.GetId(), err)
	}
}

func (m *manager) GetConnection(clusterID string) SensorConnection {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	conn := m.connectionsByClusterID[clusterID].connection
	if conn == nil {
		return nil
	}
	return conn
}

func (m *manager) replaceConnection(ctx context.Context, cluster *storage.Cluster, newConnection *sensorConnection) (oldConnection *sensorConnection, err error) {
	clusterID := cluster.GetId()
	m.connectionsByClusterIDMutex.Lock()
	defer m.connectionsByClusterIDMutex.Unlock()

	connAndUpgradeCtrl := m.connectionsByClusterID[clusterID]
	oldConnection = connAndUpgradeCtrl.connection
	if oldConnection != nil {
		if err := common.CheckConnReplace(newConnection.sensorHello.GetDeploymentIdentification(), oldConnection.sensorHello.GetDeploymentIdentification()); err != nil {
			return nil, errors.Wrapf(err, "replacing connection for cluster: %s", cluster.GetName())
		}
	}

	if err := m.clusters.UpdateSensorDeploymentIdentification(ctx, clusterID, newConnection.sensorHello.GetDeploymentIdentification()); err != nil {
		return nil, errors.Wrap(err, "updating deployment identification")
	}

	upgradeCtrl := connAndUpgradeCtrl.upgradeCtrl

	if upgradeCtrl == nil {
		upgradeCtrl, err = upgradecontroller.New(clusterID, m.clusters, m.autoTriggerUpgrades)
		if err != nil {
			return nil, err
		}
	}
	upgradeCtrlErrSig := upgradeCtrl.RegisterConnection(ctx, newConnection)
	if upgradeCtrlErrSig != nil {
		go newConnection.stopSig.SignalWhen(upgradeCtrlErrSig, concurrency.Never())
	}
	m.connectionsByClusterID[clusterID] = connectionAndUpgradeController{
		connection:  newConnection,
		upgradeCtrl: upgradeCtrl,
	}

	return oldConnection, nil
}

// CloseConnection is only used when deleting a cluster hence the removal of the deduper
func (m *manager) CloseConnection(clusterID string) {
	m.rateLimitMgr.RemoveMsgRateCluster(clusterID)

	if conn := m.GetConnection(clusterID); conn != nil {
		conn.Terminate(errors.New("cluster was deleted"))
		if !concurrency.WaitWithTimeout(conn.Stopped(), connectionTerminationTimeout) {
			utils.Should(errors.Errorf("connection to sensor from cluster %s not terminated after %v", clusterID, connectionTerminationTimeout))
		}
	}

	ctx := sac.WithAllAccess(context.Background())
	if err := m.manager.Delete(ctx, clusterID); err != nil {
		log.Errorf("deleting cluster id %q from hash manager: %v", clusterID, err)
	}
}

func (m *manager) HandleConnection(ctx context.Context, sensorHello *central.SensorHello, cluster *storage.Cluster, eventPipeline pipeline.ClusterPipeline, server central.SensorService_CommunicateServer) error {
	clusterID := cluster.GetId()
	clusterName := cluster.GetName()

	if !m.rateLimitMgr.AddInitSync(clusterID) {
		return errors.Wrap(errox.ResourceExhausted, "Central has reached the maximum number of allowed Sensors in init sync state")
	}

	conn :=
		newConnection(
			ctx,
			sensorHello,
			cluster,
			eventPipeline,
			m.clusters,
			m.networkEntities,
			m.policies,
			m.baselines,
			m.networkBaselines,
			m.delegatedRegistryConfigMgr,
			m.imageIntegrationMgr,
			m.manager,
			m.complianceOperatorMgr,
			m.rateLimitMgr,
		)
	ctx = withConnection(ctx, conn)

	oldConnection, err := m.replaceConnection(ctx, cluster, conn)
	if err != nil {
		log.Errorf("Replacing connection: %v", err)
		m.rateLimitMgr.RemoveMsgRateCluster(clusterID)
		return errors.Wrap(err, "replacing old connection")
	}

	if oldConnection != nil {
		nodeName := sensorHello.GetDeploymentIdentification().GetK8SNodeName()
		oldConnection.Terminate(errors.Errorf("a new connection for cluster %s was detected from node with name %s", clusterName, nodeName))
	}

	err = conn.Run(ctx, server, conn.capabilities)
	log.Warnf("Connection to server in cluster %s terminated: %v", clusterID, err)

	// Address the scenario in which the sensor loses its connection during
	// the initial synchronization process.
	m.rateLimitMgr.RemoveMsgRateCluster(clusterID)

	concurrency.WithLock(&m.connectionsByClusterIDMutex, func() {
		connAndUpgradeCtrl := m.connectionsByClusterID[clusterID]
		if connAndUpgradeCtrl.connection == conn {
			connAndUpgradeCtrl.connection = nil
			m.connectionsByClusterID[clusterID] = connAndUpgradeCtrl
		}
	})

	return err
}

func (m *manager) getOrCreateUpgradeCtrl(clusterID string) (upgradecontroller.UpgradeController, error) {
	m.connectionsByClusterIDMutex.Lock()
	defer m.connectionsByClusterIDMutex.Unlock()

	connAndUpgradeCtrl := m.connectionsByClusterID[clusterID]
	if connAndUpgradeCtrl.upgradeCtrl == nil {
		var err error
		connAndUpgradeCtrl.upgradeCtrl, err = upgradecontroller.New(clusterID, m.clusters, m.autoTriggerUpgrades)
		if err != nil {
			return nil, err
		}
		m.connectionsByClusterID[clusterID] = connAndUpgradeCtrl
	}
	return connAndUpgradeCtrl.upgradeCtrl, nil
}

func (m *manager) ProcessCheckInFromUpgrader(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	if err := checkClusterWriteAccess(ctx, clusterID); err != nil {
		return nil, err
	}
	upgradeCtrl, err := m.getOrCreateUpgradeCtrl(clusterID)
	if err != nil {
		return nil, err
	}
	return upgradeCtrl.ProcessCheckInFromUpgrader(req)
}

func (m *manager) ProcessUpgradeCheckInFromSensor(ctx context.Context, clusterID string, req *central.UpgradeCheckInFromSensorRequest) error {
	if err := checkClusterWriteAccess(ctx, clusterID); err != nil {
		return err
	}
	upgradeCtrl, err := m.getOrCreateUpgradeCtrl(clusterID)
	if err != nil {
		return err
	}
	return upgradeCtrl.ProcessCheckInFromSensor(req)
}

func (m *manager) checkClusterWriteAccessAndRetrieveUpgradeCtrl(ctx context.Context, clusterID string) (upgradecontroller.UpgradeController, error) {
	if err := checkClusterWriteAccess(ctx, clusterID); err != nil {
		return nil, err
	}

	upgradeCtrl := concurrency.WithRLock1(&m.connectionsByClusterIDMutex, func() upgradecontroller.UpgradeController {
		return m.connectionsByClusterID[clusterID].upgradeCtrl
	})
	if upgradeCtrl == nil {
		return nil, errors.Errorf("no upgrade controller found for cluster ID %s; either the sensor has not checked in or the clusterID is invalid. Cannot trigger upgrade", clusterID)
	}
	return upgradeCtrl, nil
}

func (m *manager) TriggerUpgrade(ctx context.Context, clusterID string) error {
	upgradeCtrl, err := m.checkClusterWriteAccessAndRetrieveUpgradeCtrl(ctx, clusterID)
	if err != nil {
		return err
	}
	return upgradeCtrl.Trigger(ctx)
}

func (m *manager) TriggerCertRotation(ctx context.Context, clusterID string) error {
	upgradeCtrl, err := m.checkClusterWriteAccessAndRetrieveUpgradeCtrl(ctx, clusterID)
	if err != nil {
		return err
	}
	return upgradeCtrl.TriggerCertRotation(ctx)
}

func (m *manager) GetActiveConnections() []SensorConnection {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	result := make([]SensorConnection, 0, len(m.connectionsByClusterID))

	for _, connAndUpgradeCtrl := range m.connectionsByClusterID {
		if conn := connAndUpgradeCtrl.connection; conn != nil {
			result = append(result, conn)
		}
	}

	return result
}

// PreparePoliciesAndBroadcast prepares and sends PolicySync message
// separately for each sensor.
func (m *manager) PreparePoliciesAndBroadcast(policies []*storage.Policy) {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	for clusterID, connAndUpgradeCtrl := range m.connectionsByClusterID {
		if connAndUpgradeCtrl.connection == nil {
			log.Debugf("could not broadcast message to cluster %q which has no active connection", clusterID)
			continue
		}

		// Downgrade policies based on the target sensor's supported version.
		msg, err := connAndUpgradeCtrl.connection.getPolicySyncMsgFromPolicies(policies)
		if err != nil {
			log.Errorf("error getting policy sync msg for cluster %q: %v", clusterID, err)
			continue
		}

		if err := connAndUpgradeCtrl.connection.InjectMessage(concurrency.Never(), msg); err != nil {
			log.Errorf("error broadcasting message to cluster %q", clusterID)
		}
	}

}

func (m *manager) BroadcastMessage(msg *central.MsgToSensor) {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	for clusterID, connAndUpgradeCtrl := range m.connectionsByClusterID {
		if connAndUpgradeCtrl.connection == nil {
			log.Debugf("could not broadcast message to cluster %q which has no active connection", clusterID)
			continue
		}
		if err := connAndUpgradeCtrl.connection.InjectMessage(concurrency.Never(), msg); err != nil {
			log.Errorf("error broadcasting message to cluster %q", clusterID)
		}
	}
}

func (m *manager) SendMessage(clusterID string, msg *central.MsgToSensor) error {
	m.connectionsByClusterIDMutex.RLock()
	defer m.connectionsByClusterIDMutex.RUnlock()

	connAndUpgradeCtrl, ok := m.connectionsByClusterID[clusterID]
	if !ok {
		return errors.Errorf("no cluster %q connection exists", clusterID)
	}
	if connAndUpgradeCtrl.connection == nil {
		return errors.Errorf("no valid cluster %q connection", clusterID)
	}
	return connAndUpgradeCtrl.connection.InjectMessage(concurrency.Never(), msg)
}

func (m *manager) PushExternalNetworkEntitiesToSensor(ctx context.Context, clusterID string) error {
	conn := m.GetConnection(clusterID)
	if conn == nil {
		return nil
	}

	// This is not perfect, however, the closest.
	if !conn.HasCapability(centralsensor.NetworkGraphExternalSrcsCap) {
		return errors.New("sensor version must be up-to-date with Central")
	}
	return conn.NetworkEntities().SyncNow(ctx)
}

func (m *manager) PushExternalNetworkEntitiesToAllSensors(ctx context.Context) error {
	var errs errorhelpers.ErrorList
	for _, conn := range m.GetActiveConnections() {
		// This is not perfect, however, the closest.
		if !conn.HasCapability(centralsensor.NetworkGraphExternalSrcsCap) {
			errs.AddError(errors.Errorf("sensor version for cluster %q is not up-to-date with Central", conn.ClusterID()))
			continue
		}

		if err := conn.NetworkEntities().SyncNow(ctx); err != nil {
			errs.AddError(err)
		}
	}
	return errs.ToError()
}
