package manager

import (
	"context"
	"errors"
	"time"

	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/paladin"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

const (
	discoveredClustersLoopInterval = 10 * time.Minute
)

var (
	_ Manager = (*managerImpl)(nil)

	cloudSourceCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Integration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		))

	log = logging.LoggerForModule()
)

type managerImpl struct {
	shortCircuitSignal concurrency.Signal
	stopSignal         concurrency.Signal

	loopInterval time.Duration
	loopTicker   *time.Ticker

	cloudSourcesDataStore cloudSourcesDS.DataStore
}

func newManager(datastore cloudSourcesDS.DataStore) *managerImpl {
	return &managerImpl{
		shortCircuitSignal:    concurrency.NewSignal(),
		stopSignal:            concurrency.NewSignal(),
		loopInterval:          discoveredClustersLoopInterval,
		cloudSourcesDataStore: datastore,
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
			m.getAssetsFromCloudSources()
		case <-m.loopTicker.C:
			m.getAssetsFromCloudSources()
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *managerImpl) getAssetsFromCloudSources() {
	// Fetch the cloud sources from the datastore. This will ensure that we will always use the latest updates.
	// For the time being we do not foresee this to be a high cardinality store.
	cloudSources, err := m.cloudSourcesDataStore.ListCloudSources(cloudSourceCtx, search.EmptyQuery())
	if err != nil {
		log.Errorw("Failed listing stored cloud sources", logging.Err(err))
		return
	}

	clients := createClients(cloudSources)

	var clientErrors error
	var responses []*paladin.AssetsResponse
	for _, client := range clients {
		resp, err := client.GetAssets(context.Background())
		if err != nil {
			clientErrors = errors.Join(clientErrors, err)
			continue
		}
		responses = append(responses, resp)
	}

	if clientErrors != nil {
		log.Errorw("Received errors during fetching assets from cloud sources. The result might be incomplete",
			logging.Err(clientErrors))
	}

	// TODO: Once the discovered clusters are available, transform the assets response to discovered clusters
	//       (or rather, move this to the client and create a generic interface for that).
	log.Infof("Got the following assets from Cloud Source integrations: %v", responses)
}

// createClients creates the API clients to interact with the third-party API of the cloud source.
// For the time being, this is Paladin Cloud only.
func createClients(cloudSources []*storage.CloudSource) []*paladin.Client {
	clients := make([]*paladin.Client, 0, len(cloudSources))
	for _, cloudSource := range cloudSources {
		clients = append(clients, paladin.NewClient(cloudSource))
	}
	return clients
}
