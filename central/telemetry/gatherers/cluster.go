package gatherers

import (
	"context"
	"encoding/json"
	"time"

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

// ClusterGatherer gathers a list of telemetry info for all clusters
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
// the latest info for offline sensors using the given datastores
func newClusterGatherer(clusterDatastore clusterDatastore.DataStore, nodeDatastore nodeDatastore.DataStore, namespaceDatastore namespaceDatastore.DataStore, sensorConnMgr connection.Manager, deploymentDatastore deploymentDatastore.DataStore) *ClusterGatherer {
	nodeGatherer := newNodeGatherer(nodeDatastore)
	namespaceGatherer := newNamespaceGatherer(namespaceDatastore, deploymentDatastore)
	return &ClusterGatherer{
		clusterDatastore:  clusterDatastore,
		nodeGatherer:      nodeGatherer,
		namespaceGatherer: namespaceGatherer,
		sensorConnMgr:     sensorConnMgr,
	}
}

// Gather returns a list of stats about all the clusters monitored by this StackRox installation
func (c *ClusterGatherer) Gather(ctx context.Context, pullFromSensors bool) []*data.ClusterInfo {
	var clusterList []*data.ClusterInfo

	clusters, err := c.clusterDatastore.GetClusters(ctx)
	if err != nil {
		log.Errorf("Unable to load clusters for telemetry: %v", err)
		return nil
	}
	clusterMap := make(map[string]*storage.Cluster, len(clusters))
	for _, cluster := range clusters {
		clusterMap[cluster.GetId()] = cluster
	}

	// Get active clusters
	clusterRetC := make(chan clusterFromSensorResponse)
	outstanding := 0

	if pullFromSensors {
		for _, sensorConn := range c.sensorConnMgr.GetActiveConnections() {
			go c.clusterFromSensor(ctx, sensorConn, clusterRetC, clusterMap)
			outstanding++
		}
	}

	for outstanding > 0 {
		select {
		case <-ctx.Done():
			outstanding = 0 // proceed with partial data

		case ret := <-clusterRetC:
			outstanding--
			if ret.err != nil {
				log.Errorf("Error pulling telemetry data from cluster %s: %v", ret.clusterID, ret.err)
				continue
			}
			clusterList = append(clusterList, ret.clusterInfo)
			delete(clusterMap, ret.clusterID)
		}
	}

	// Get inactive clusters
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

func (c *ClusterGatherer) clusterFromSensor(ctx context.Context, sensorConn connection.SensorConnection, outC chan<- clusterFromSensorResponse, clusterMap map[string]*storage.Cluster) {
	clusterInfo, err := c.fetchClusterFromSensor(ctx, sensorConn, clusterMap)
	if err != nil {
		log.Warnf("Error pulling cluster info from sensor: %w", err)
	}
	select {
	case <-ctx.Done():
	case outC <- clusterFromSensorResponse{
		clusterID:   sensorConn.ClusterID(),
		clusterInfo: clusterInfo,
		err:         err,
	}:
	}
}

func (c *ClusterGatherer) fetchClusterFromSensor(ctx context.Context, sensorConn connection.SensorConnection, clusterMap map[string]*storage.Cluster) (*data.ClusterInfo, error) {
	var clusterBytes []byte
	callback := func(ctx concurrency.ErrorWaitable, sensorInfo *central.TelemetryResponsePayload_ClusterInfo) error {
		clusterBytes = append(clusterBytes, sensorInfo.Chunk...)
		return nil
	}

	pullClusterCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := sensorConn.Telemetry().PullClusterInfo(pullClusterCtx, callback)
	if err != nil {
		return nil, err
	}

	var cluster data.ClusterInfo
	err = json.Unmarshal(clusterBytes, &cluster)
	if err != nil {
		return nil, err
	}
	cluster.ID = sensorConn.ClusterID()
	cluster.HelmManaged = clusterMap[cluster.ID].GetHelmConfig() != nil
	if cluster.Sensor != nil {
		curTime := time.Now()
		cluster.Sensor.LastCheckIn = &curTime
	}
	return &cluster, nil
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
