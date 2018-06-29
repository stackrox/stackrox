package datastore

import (
	"time"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/cluster/store"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	dnrDataStore "bitbucket.org/stack-rox/apollo/central/dnrintegration/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetCluster(id string) (*v1.Cluster, bool, error)
	GetClusters() ([]*v1.Cluster, error)
	CountClusters() (int, error)

	AddCluster(cluster *v1.Cluster) (string, error)
	UpdateCluster(cluster *v1.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
}

// New returns an instance of DataStore.
func New(storage store.Store, ads alertDataStore.DataStore, dds deploymentDataStore.DataStore, dnr dnrDataStore.DataStore) DataStore {
	return &datastoreImpl{
		storage: storage,
		ads:     ads,
		dds:     dds,
		dnr:     dnr,
	}
}
