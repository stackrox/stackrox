package gatherers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	concPool "github.com/sourcegraph/conc/pool"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

// ClusterGatherer gathers a list of telemetry info for all clusters.
type ClusterGatherer struct {
	clusterDatastore  clusterDatastore.DataStore
	nodeGatherer      *nodeGatherer
	namespaceGatherer *namespaceGatherer
	sensorConnMgr     connection.Manager
}

var (
	log = logging.LoggerForModule()
)

// newClusterGatherer returns a new ClusterGatherer which will query connected Sensors for telemetry info and collect
// the latest info for offline sensors using the given datastores.
func newClusterGatherer(clusterDatastore clusterDatastore.DataStore, nodeDatastore nodeDatastore.DataStore,
	namespaceDatastore namespaceDatastore.DataStore, sensorConnMgr connection.Manager,
	deploymentDatastore deploymentDatastore.DataStore) *ClusterGatherer {
	nodeGatherer := newNodeGatherer(nodeDatastore)
	namespaceGatherer := newNamespaceGatherer(namespaceDatastore, deploymentDatastore)
	return &ClusterGatherer{
		clusterDatastore:  clusterDatastore,
		nodeGatherer:      nodeGatherer,
		namespaceGatherer: namespaceGatherer,
		sensorConnMgr:     sensorConnMgr,
	}
}

// Gather returns a list of stats about all the clusters monitored by this StackRox installation.
func (c *ClusterGatherer) Gather(ctx context.Context, pullFromSensors bool) []*data.ClusterInfo {
	var clusterList []*data.ClusterInfo

	clusters, err := c.clusterDatastore.GetClusters(ctx)
	if err != nil {
		log.Errorw("Unable to load clusters for telemetry", logging.Err(err))
		return nil
	}
	clusterMap := make(map[string]*storage.Cluster, len(clusters))
	for _, cluster := range clusters {
		clusterMap[cluster.GetId()] = cluster
	}

	gatherPool := concPool.NewWithResults[clusterFromSensorResponse]().WithContext(ctx)
	if pullFromSensors {
		for _, conn := range c.sensorConnMgr.GetActiveConnections() {
			cluster := clusterMap[conn.ClusterID()]
			gatherPool.Go(func(ctx context.Context) (clusterFromSensorResponse, error) {
				return c.clusterFromSensor(ctx, conn, cluster), nil
			})
		}
	}

	var gatherResults []clusterFromSensorResponse
	if ctxErr := concurrency.DoInWaitable(ctx, func() {
		gatherResults, _ = gatherPool.Wait()
	}); ctxErr != nil {
		log.Errorw("Context error during pulling telemetry data from clusters", logging.Err(err))
		return nil
	}

	for _, result := range gatherResults {
		if result.err != nil {
			log.Errorw("Error pulling telemetry data from cluster", logging.ClusterID(result.clusterID),
				logging.Err(result.err))
			continue
		}
		clusterList = append(clusterList, result.clusterInfo)
		delete(clusterMap, result.clusterID)
	}

	// Get inactive clusters.
	for _, storageCluster := range clusterMap {
		clusterList = append(clusterList, c.clusterFromDatastores(ctx, storageCluster))
	}

	return clusterList
}

type clusterFromSensorResponse struct {
	clusterID   string
	clusterInfo *data.ClusterInfo
	err         error
}

func (c *ClusterGatherer) clusterFromSensor(ctx context.Context, sensorConn connection.SensorConnection,
	cluster *storage.Cluster) clusterFromSensorResponse {
	clusterInfo, err := c.fetchClusterFromSensor(ctx, sensorConn, cluster)
	return clusterFromSensorResponse{
		clusterID:   sensorConn.ClusterID(),
		clusterInfo: clusterInfo,
		err:         err,
	}
}

func (c *ClusterGatherer) fetchClusterFromSensor(ctx context.Context, sensorConn connection.SensorConnection,
	cluster *storage.Cluster) (*data.ClusterInfo, error) {
	var clusterBytes []byte
	callback := func(ctx concurrency.ErrorWaitable, sensorInfo *central.TelemetryResponsePayload_ClusterInfo) error {
		clusterBytes = append(clusterBytes, sensorInfo.Chunk...)
		return nil
	}

	clusterID := sensorConn.ClusterID()
	err := sensorConn.Telemetry().PullClusterInfo(ctx, callback)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to pull cluster info for the cluster %s", cluster.GetName())
	}

	var clusterInfo data.ClusterInfo
	err = json.Unmarshal(clusterBytes, &cluster)
	if err != nil {
		return nil, err
	}
	clusterInfo.ID = clusterID
	clusterInfo.HelmManaged = cluster.GetHelmConfig() != nil
	if clusterInfo.Sensor != nil {
		curTime := time.Now()
		clusterInfo.Sensor.LastCheckIn = &curTime
	}
	return &clusterInfo, nil
}

func (c *ClusterGatherer) clusterFromDatastores(ctx context.Context, cluster *storage.Cluster) *data.ClusterInfo {
	status := cluster.GetStatus()
	lastContact := getLastContact(cluster)
	orchestrator := status.GetOrchestratorMetadata()
	providerMetadata := status.GetProviderMetadata()
	namespaces, errList := c.namespaceGatherer.Gather(ctx, cluster.GetId())
	clusterInfo := &data.ClusterInfo{
		ID:          cluster.GetId(),
		HelmManaged: cluster.GetHelmConfig() != nil,
		Sensor: &data.SensorInfo{
			RoxComponentInfo: &data.RoxComponentInfo{
				Version:  status.GetSensorVersion(),
				Process:  nil,
				Restarts: 0,
			},
			LastCheckIn:        lastContact,
			CurrentlyConnected: false,
		},
		Orchestrator: &data.OrchestratorInfo{
			Orchestrator:        cluster.GetType().String(),
			OrchestratorVersion: orchestrator.GetVersion(),
		},
		Nodes:         c.nodeGatherer.Gather(ctx, cluster.GetId()),
		Namespaces:    namespaces,
		CloudProvider: telemetry.GetProviderString(providerMetadata),
		Errors:        errList,
	}
	return clusterInfo
}

func getLastContact(cluster *storage.Cluster) *time.Time {
	return telemetry.GetTimeOrNil(cluster.GetHealthStatus().GetLastContact())
}
