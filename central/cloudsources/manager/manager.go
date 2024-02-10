package manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/time/rate"
)

const (
	discoveredClustersLoopInterval = 10 * time.Minute
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
	discoveredClustersDS discoveredClustersDS.DataStore, clustersDS clusterDS.DataStore) *managerImpl {
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
	cloudSources, err := m.cloudSourcesDataStore.ListCloudSources(cloudSourceCtx, search.EmptyQuery())
	if err != nil {
		log.Errorw("Failed listing stored cloud sources", logging.Err(err))
		return nil
	}

	clients := createClients(cloudSources)

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
	m.matchDiscoveredClusters(clusters)

	debugPrintDiscoveredClusters(clusters)

	if err := m.discoveredClustersDataStore.UpsertDiscoveredClusters(discoveredClusterCtx, clusters...); err != nil {
		log.Errorw("Received errors during upserting discovered clusters.", logging.Err(err))
	}
}

func (m *managerImpl) matchDiscoveredClusters(clusters []*discoveredclusters.DiscoveredCluster) {
	// A list of hashes of currently secured cluster. A secured cluster hash consist of:
	//  The secured cluster ID, and type.
	securedClusters := set.NewStringSet()

	unspecifiedProviderType := set.NewStringSet()

	if err := m.clusterDataStore.WalkClusters(clustersCtx, func(obj *storage.Cluster) error {
		providerMetadata := obj.GetStatus().GetProviderMetadata()

		// Explicitly ignore clusters which do not have provider metadata associated with them.
		if providerMetadata == nil {
			return nil
		}

		// In case we have partial provider metadata (e.g. we miss the cluster name for EKS clusters, or the
		// cluster type is unspecified), then mark the provider type as Unspecified.
		// This means that we will assign each discovered cluster that cannot be safely matched to a secured cluster
		// as Unspecified instead of Unsecured. This is to avoid false-positives.
		clusterMetadata := providerMetadata.GetCluster()
		if clusterMetadata.GetId() == "" {
			unspecifiedProviderType.Add(providerMetadataToProviderType(providerMetadata).String())
			return nil
		}

		// Add the cluster with the full metadata information we require as an index for matching.
		securedClusters.Add(clusterIndexForClusterMetadata(clusterMetadata))
		return nil
	}); err != nil {
		log.Errorw("Failed to list secured clusters. Matching is skipped.", logging.Err(err))
		return
	}

	for _, cluster := range clusters {
		switch {
		case securedClusters.Contains(clusterIndexForDiscoveredCluster(cluster)):
			cluster.Status = storage.DiscoveredCluster_STATUS_SECURED
		case unspecifiedProviderType.Contains(cluster.GetProviderType().String()):
			cluster.Status = storage.DiscoveredCluster_STATUS_UNSPECIFIED
		default:
			cluster.Status = storage.DiscoveredCluster_STATUS_UNSECURED
		}
	}
}

// createClients creates the API clients to interact with the third-party API of the cloud source.
func createClients(cloudSources []*storage.CloudSource) []cloudsources.Client {
	clients := make([]cloudsources.Client, 0, len(cloudSources))
	for _, cloudSource := range cloudSources {
		if client := cloudsources.NewClientForCloudSource(cloudSource); client != nil {
			clients = append(clients, client)
		}
	}
	return clients
}

type clusterIndex = string

func clusterIndexForClusterMetadata(obj *storage.ClusterMetadata) clusterIndex {
	return obj.GetId() + obj.GetType().String()
}

func clusterIndexForDiscoveredCluster(obj *discoveredclusters.DiscoveredCluster) clusterIndex {
	return obj.GetID() + obj.GetType().String()
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
