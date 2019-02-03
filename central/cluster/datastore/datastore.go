package datastore

import (
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/streamer"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetCluster(id string) (*storage.Cluster, bool, error)
	GetClusters() ([]*storage.Cluster, error)
	CountClusters() (int, error)

	AddCluster(cluster *storage.Cluster) (string, error)
	UpdateCluster(cluster *storage.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
	UpdateMetadata(id string, metadata *storage.ProviderMetadata) error

	Search(q *v1.Query) ([]search.Result, error)
}

// New returns an instance of DataStore.
func New(
	storage store.Store,
	indexer index.Indexer,
	ads alertDataStore.DataStore,
	dds deploymentDataStore.DataStore,
	ns nodeStore.GlobalStore,
	ss secretDataStore.DataStore,
	sm streamer.Manager) (DataStore, error) {
	ds := &datastoreImpl{
		storage: storage,
		indexer: indexer,
		ads:     ads,
		dds:     dds,
		ns:      ns,
		ss:      ss,
		sm:      sm,
	}
	if err := ds.buildIndex(); err != nil {
		return ds, err
	}
	return ds, nil
}
