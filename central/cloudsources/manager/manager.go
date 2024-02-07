package manager

import (
	"context"
	"errors"
	"time"

	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

const (
	discoveredClustersLoopInterval = 10 * time.Minute

	discoveredClustersRetentionTime = 24 * time.Hour
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

	log = logging.LoggerForModule()
)

type managerImpl struct {
	shortCircuitSignal concurrency.Signal
	stopSignal         concurrency.Signal

	loopInterval time.Duration
	loopTicker   *time.Ticker

	cloudSourcesDataStore       cloudSourcesDS.DataStore
	discoveredClustersDataStore discoveredClustersDS.DataStore
}

func newManager(cloudSourcesDS cloudSourcesDS.DataStore,
	discoveredClustersDS discoveredClustersDS.DataStore) *managerImpl {
	return &managerImpl{
		shortCircuitSignal:          concurrency.NewSignal(),
		stopSignal:                  concurrency.NewSignal(),
		loopInterval:                discoveredClustersLoopInterval,
		cloudSourcesDataStore:       cloudSourcesDS,
		discoveredClustersDataStore: discoveredClustersDS,
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
	log.Debugf("Got the following assets from Cloud Source integrations: %+v", discoveredClusters)
	return discoveredClusters
}

func (m *managerImpl) reconcileDiscoveredClusters(clusters []*discoveredclusters.DiscoveredCluster) {
	// Remove discovered clusters which are out of the retention time.
	deleted, err := m.discoveredClustersDataStore.DeleteDiscoveredClusters(discoveredClusterCtx, search.NewQueryBuilder().
		AddTimeRangeField(
			search.LastUpdatedTime,
			time.Unix(0, 0),
			time.Now().Add(-discoveredClustersRetentionTime),
		).ProtoQuery())
	if err != nil {
		log.Errorw("Received errors during deletion of discovered clusters who haven't been updated outside of the retention time",
			logging.Err(err))
	} else {
		log.Debugf("Deleted %d discovered clusters due to being outside of the retention time", len(deleted))
	}

	// TODO: Add matching of discovered clusters with secured clusters.

	if err := m.discoveredClustersDataStore.UpsertDiscoveredClusters(discoveredClusterCtx, clusters...); err != nil {
		log.Errorw("Received errors during upserting discovered clusters.", logging.Err(err))
	}
}

// createClients creates the API clients to interact with the third-party API of the cloud source.
func createClients(cloudSources []*storage.CloudSource) []cloudsources.Client {
	clients := make([]cloudsources.Client, 0, len(cloudSources))
	for _, cloudSource := range cloudSources {
		clients = append(clients, cloudsources.NewClientForCloudSource(cloudSource))
	}
	return clients
}
