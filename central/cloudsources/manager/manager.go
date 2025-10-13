package manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgErrors "github.com/pkg/errors"
	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/storagetotype"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/time/rate"
)

const (
	discoveredClustersLoopInterval = 10 * time.Minute
	clientCreationTimeout          = 30 * time.Second
)

var (
	_ Manager = (*managerImpl)(nil)

	cloudSourceCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Integration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		))

	discoveredClusterCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Administration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		))

	clustersCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Cluster),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		))

	log = logging.LoggerForModule()
)

type managerImpl struct {
	shortCircuitSignal concurrency.Signal
	stopSignal         concurrency.Signal

	loopInterval            time.Duration
	loopTicker              *time.Ticker
	shortCircuitRateLimiter *rate.Limiter

	cloudSourcesDataStore       cloudSourcesDS.DataStore
	discoveredClustersDataStore discoveredClustersDS.DataStore
	clusterDataStore            clusterDS.DataStore
}

func newManager(cloudSourcesDS cloudSourcesDS.DataStore,
	discoveredClustersDS discoveredClustersDS.DataStore, clustersDS clusterDS.DataStore,
) *managerImpl {
	return &managerImpl{
		shortCircuitSignal:          concurrency.NewSignal(),
		stopSignal:                  concurrency.NewSignal(),
		loopInterval:                discoveredClustersLoopInterval,
		shortCircuitRateLimiter:     rate.NewLimiter(rate.Every(time.Minute), 1),
		cloudSourcesDataStore:       cloudSourcesDS,
		discoveredClustersDataStore: discoveredClustersDS,
		clusterDataStore:            clustersDS,
	}
}

// Start the collection of assets from cloud sources.
func (m *managerImpl) Start() {
	m.loopTicker = time.NewTicker(m.loopInterval)
	go m.discoveredClustersLoop()
}

// Stop the collection of assets from cloud sources.
func (m *managerImpl) Stop() {
	m.stopSignal.Signal()
}

// ShortCircuit the collection of assets from cloud sources.
func (m *managerImpl) ShortCircuit() {
	m.shortCircuitSignal.Signal()
}

func (m *managerImpl) MarkClusterSecured(id string) {
	log.Infof("Marking discovered clusters matching cluster %q as secured", id)
	m.changeStatusForDiscoveredClusters(id, storage.DiscoveredCluster_STATUS_SECURED)
}

func (m *managerImpl) MarkClusterUnsecured(id string) {
	log.Infof("Marking discovered clusters matching cluster %q as unsecured", id)
	m.changeStatusForDiscoveredClusters(id, storage.DiscoveredCluster_STATUS_UNSECURED)
}

func (m *managerImpl) discoveredClustersLoop() {
	defer m.loopTicker.Stop()

	for {
		select {
		case <-m.shortCircuitSignal.Done():
			if err := m.shortCircuitRateLimiter.Wait(concurrency.AsContext(&m.stopSignal)); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				log.Errorw("Waiting for rate limiter entrance", logging.Err(err))
			}
			// Make sure to reset the signal again.
			m.shortCircuitSignal.Reset()
			// Make sure to reset the ticker, so we are not in the case where short-circuit is called and shortly after
			// the interval is reached and discovered clusters are gathered again.
			m.loopTicker.Reset(m.loopInterval)
			m.discoverClustersFromCloudSources()
		case <-m.loopTicker.C:
			m.discoverClustersFromCloudSources()
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *managerImpl) discoverClustersFromCloudSources() {
	clusters := m.getDiscoveredClustersFromCloudSources()

	m.reconcileDiscoveredClusters(clusters)
}

func (m *managerImpl) getDiscoveredClustersFromCloudSources() []*discoveredclusters.DiscoveredCluster {
	// Fetch the cloud sources from the datastore. This will ensure that we will always use the latest updates.
	// For the time being we do not foresee this to be a high cardinality store.
	createCtx, cancel := context.WithTimeout(context.Background(), clientCreationTimeout)
	defer cancel()

	var clients []cloudsources.Client
	var clientCreationErrs error
	err := m.cloudSourcesDataStore.ForEachCloudSource(cloudSourceCtx, func(cloudSource *storage.CloudSource) error {
		client, err := cloudsources.NewClientForCloudSource(createCtx, cloudSource)
		if err != nil {
			clientCreationErrs = errors.Join(clientCreationErrs,
				pkgErrors.Wrapf(err, "creating client for cloud source %q", cloudSource.GetName()))
		}
		if client != nil {
			clients = append(clients, client)
		}
		return nil
	})
	if err != nil {
		log.Errorw("Failed listing stored cloud sources", logging.Err(err))
		return nil
	}

	if clientCreationErrs != nil {
		log.Errorw("Received errors during creating clients from cloud sources. The result might be incomplete",
			logging.Err(clientCreationErrs))
	}

	var clientErrors error
	var discoveredClusters []*discoveredclusters.DiscoveredCluster
	for _, client := range clients {
		resp, err := client.GetDiscoveredClusters(context.Background())
		if err != nil {
			clientErrors = errors.Join(clientErrors, err)
			continue
		}
		discoveredClusters = append(discoveredClusters, resp...)
	}

	if clientErrors != nil {
		log.Errorw("Received errors during fetching assets from cloud sources. The result might be incomplete",
			logging.Err(clientErrors))
	}
	debugPrintDiscoveredClusters(discoveredClusters)
	return discoveredClusters
}

func (m *managerImpl) reconcileDiscoveredClusters(clusters []*discoveredclusters.DiscoveredCluster) {
	log.Info("Fetching discovered clusters from cloud sources")
	m.matchDiscoveredClusters(clusters)

	log.Infof("Received %d discovered clusters from cloud sources", len(clusters))
	debugPrintDiscoveredClusters(clusters)

	if err := m.discoveredClustersDataStore.UpsertDiscoveredClusters(discoveredClusterCtx, clusters...); err != nil {
		log.Errorw("Received errors during upserting discovered clusters.", logging.Err(err))
	}
}

func (m *managerImpl) matchDiscoveredClusters(clusters []*discoveredclusters.DiscoveredCluster) {
	// A list of IDs of secured clusters that can be used for matching.
	securedClusters := set.NewStringSet()

	unspecifiedProviderTypes := set.NewStringSet()

	if err := m.clusterDataStore.WalkClusters(clustersCtx, func(obj *storage.Cluster) error {
		// Explicitly ignore unhealthy clusters for the matching, as they cannot be deemed as secured.
		if obj.GetHealthStatus().GetOverallHealthStatus() != storage.ClusterHealthStatus_HEALTHY {
			return nil
		}

		providerMetadata := obj.GetStatus().GetProviderMetadata()

		// Explicitly ignore clusters which do not have provider metadata associated with them.
		if providerMetadata == nil {
			return nil
		}

		// In case we have partial provider metadata (i.e. the ID of the cluster isn't set), then mark the provider
		// type as Unspecified.
		// This means that we will assign each discovered cluster that cannot be safely matched to a secured cluster
		// as Unspecified instead of Unsecured. This is to avoid false-positives.
		clusterMetadata := providerMetadata.GetCluster()
		if clusterMetadata.GetId() == "" {
			unspecifiedProviderTypes.Add(providerMetadataToProviderType(providerMetadata).String())
			return nil
		}

		// Add the cluster with the full metadata information we require as an index for matching.
		securedClusters.Add(clusterMetadata.GetId())
		return nil
	}); err != nil {
		log.Errorw("Failed to list secured clusters. Matching is skipped.", logging.Err(err))
		return
	}

	for _, cluster := range clusters {
		switch {
		case securedClusters.Contains(cluster.GetID()):
			cluster.Status = storage.DiscoveredCluster_STATUS_SECURED
		case unspecifiedProviderTypes.Contains(cluster.GetProviderType().String()):
			cluster.Status = storage.DiscoveredCluster_STATUS_UNSPECIFIED
		default:
			cluster.Status = storage.DiscoveredCluster_STATUS_UNSECURED
		}
	}
}

func (m *managerImpl) changeStatusForDiscoveredClusters(clusterID string, status storage.DiscoveredCluster_Status) {
	cluster, err := m.getCluster(clusterID)
	if err != nil {
		log.Errorw("Failed to get cluster to change status",
			logging.ClusterID(clusterID), logging.Err(err))
		return
	}

	// If the cluster has no metadata, we can short-circuit since it cannot match with any discovered cluster.
	if cluster.GetStatus().GetProviderMetadata().GetCluster().GetId() == "" {
		return
	}

	var discoveredClusterIds []string
	err = m.cloudSourcesDataStore.ForEachCloudSource(cloudSourceCtx, func(c *storage.CloudSource) error {
		discoveredClusterIds = append(discoveredClusterIds, createDiscoveredClusterId(cluster, c))
		return nil
	})
	if err != nil {
		log.Errorw("Failed to list stored cloud sources for changing cluster status",
			logging.Err(err), logging.ClusterID(clusterID))
		return
	}

	discoveredClusters := m.fetchDiscoveredClusters(discoveredClusterIds)
	// In case we found no discovered clusters, we can short-circuit here.
	if len(discoveredClusters) == 0 {
		return
	}

	for _, discoveredCluster := range discoveredClusters {
		discoveredCluster.Status = status
	}

	if err := m.discoveredClustersDataStore.UpsertDiscoveredClusters(discoveredClusterCtx,
		storagetotype.DiscoveredClusters(discoveredClusters...)...); err != nil {
		log.Errorw("Failed changing status of discovered clusters",
			logging.Err(err), logging.ClusterID(clusterID))
	}
}

func (m *managerImpl) getCluster(id string) (*storage.Cluster, error) {
	cluster, exists, err := m.clusterDataStore.GetCluster(clustersCtx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return cluster, nil
}

func (m *managerImpl) fetchDiscoveredClusters(ids []string) []*storage.DiscoveredCluster {
	discoveredClusters := make([]*storage.DiscoveredCluster, 0, len(ids))
	for _, id := range ids {
		cluster, err := m.discoveredClustersDataStore.GetDiscoveredCluster(discoveredClusterCtx, id)
		if err == nil {
			discoveredClusters = append(discoveredClusters, cluster)
		}
	}
	return discoveredClusters
}

func debugPrintDiscoveredClusters(clusters []*discoveredclusters.DiscoveredCluster) {
	logMsg := "Got the following discovered clusters:\n"
	for i, cluster := range clusters {
		logMsg = fmt.Sprintf("%s%d: %+v\n", logMsg, i, cluster)
	}
	log.Debug(logMsg)
}

func providerMetadataToProviderType(metadata *storage.ProviderMetadata) storage.DiscoveredCluster_Metadata_ProviderType {
	switch {
	case metadata.GetGoogle() != nil:
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP
	case metadata.GetAws() != nil:
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS
	case metadata.GetAzure() != nil:
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE
	default:
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_UNSPECIFIED
	}
}

func createDiscoveredClusterId(cluster *storage.Cluster, cloudSource *storage.CloudSource) string {
	return discoveredclusters.
		GenerateDiscoveredClusterID(&discoveredclusters.DiscoveredCluster{
			ID:            cluster.GetStatus().GetProviderMetadata().GetCluster().GetId(),
			CloudSourceID: cloudSource.GetId(),
		})
}
