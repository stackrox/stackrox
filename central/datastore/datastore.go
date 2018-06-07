package datastore

import (
	"net/http"
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/db/boltdb"
	"bitbucket.org/stack-rox/apollo/central/db/inmem"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/central/search/blevesearch"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()

	mutex    sync.RWMutex
	registry *DataStore
)

// DataStore provides access to datastores for reading and modifying saved data.
// It restrict access to the interfaces provided here and doesn't allow access to the lower level constructs.
type DataStore struct {
	// Base objects that underly datastores.
	inmem   db.Storage
	indexer search.Indexer

	// Datastore objects.
	alerts      AlertDataStore
	benchmarks  BenchmarkDataStore
	clusters    ClusterDataStore
	deployments DeploymentDataStore
	images      ImageDataStore
	policies    PolicyDataStore
}

// Init takes in a storage implementation and and indexer implementation
func Init() error {
	mutex.Lock()
	defer mutex.Unlock()

	if registry != nil {
		panic("datastore initialized more than once")
	}

	persistence, err := boltdb.NewWithDefaults(env.DBPath.Setting())
	if err != nil {
		return err
	}
	inmem := inmem.New(persistence)

	indexer, err := blevesearch.NewIndexer("/tmp/moss.bleve")
	if err != nil {
		return err
	}

	// Include any storage types you'd like to override.
	alerts, err := NewAlertDataStore(inmem, indexer)
	if err != nil {
		return err
	}
	benchmarks, err := NewBenchmarkDataStore(inmem)
	if err != nil {
		return err
	}
	deployments, err := NewDeploymentDataStore(inmem, indexer)
	if err != nil {
		return err
	}
	images, err := NewImageDataStore(inmem, indexer)
	if err != nil {
		return err
	}

	policies, err := NewPolicyDataStore(inmem, indexer)
	if err != nil {
		return err
	}
	clusters := NewClusterDataStore(inmem, deployments, alerts)
	if err != nil {
		return err
	}
	// Build and return the datastore.
	registry = &DataStore{
		inmem:   inmem,
		indexer: indexer,

		alerts:      alerts,
		benchmarks:  benchmarks,
		clusters:    clusters,
		deployments: deployments,
		images:      images,
		policies:    policies,
	}
	return nil
}

// Close closes the registry, which closes the underlying inmem db, and the indexer.
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	if registry != nil {
		panic("datastore closed when not open")
	}

	registry.inmem.Close()
	registry.indexer.Close()

	registry.inmem = nil
	registry.indexer = nil

	registry.alerts = nil
	registry.benchmarks = nil
	registry.clusters = nil
	registry.deployments = nil
	registry.images = nil
	registry.policies = nil
}

// Export and Backup handlers served from the inmem object.
///////////////////////////////////////////////////////////

// BackupHandler provides the http.Handler from the inderlying in memory db.
func BackupHandler() http.Handler {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.ExportHandler()
}

// ExportHandler provides the http.Handler from the inderlying in memory db.
func ExportHandler() http.Handler {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.BackupHandler()
}

// Use the registries datastore objects.
////////////////////////////////////////

// GetAlertDataStore provides an instance of AlertDataStore created by the registry.
func GetAlertDataStore() AlertDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.alerts
}

// GetBenchmarkDataStore provides an instance of BenchmarkDataStore created by the registry.
func GetBenchmarkDataStore() BenchmarkDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.benchmarks
}

// GetClusterDataStore provides an instance of ClusterDataStore created by the registry.
func GetClusterDataStore() ClusterDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.clusters
}

// GetDeploymentDataStore provides an instance of DeploymentDataStore created by the registry.
func GetDeploymentDataStore() DeploymentDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.deployments
}

// GetImageDataStore provides an instance of ImageDataStore created by the registry.
func GetImageDataStore() ImageDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.images
}

// GetPolicyDataStore provides an instance of PolicyDataStore created by the registry.
func GetPolicyDataStore() PolicyDataStore {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.policies
}

// Use the registries underlying storage object as sub-interfaces in a type-safe way.
// These are basically pass through in functionality, using the DB directly.
////////////////////////////////////////////////////////////////////////////

// GetAuthProviderStorage provide storage functionality for authProvider.
func GetAuthProviderStorage() db.AuthProviderStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.AuthProviderStorage)
}

// GetBenchmarkScheduleStorage provides storage functionality for benchmark schedules.
func GetBenchmarkScheduleStorage() db.BenchmarkScheduleStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.BenchmarkScheduleStorage)
}

// GetBenchmarkScansStorage provides storage functionality for benchmarks scans.
func GetBenchmarkScansStorage() db.BenchmarkScansStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.BenchmarkScansStorage)
}

// GetBenchmarkTriggerStorage provides storage functionality for benchmarks triggers.
func GetBenchmarkTriggerStorage() db.BenchmarkTriggerStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.BenchmarkTriggerStorage)
}

// GetImageIntegrationStorage provide storage functionality for image integrations.
func GetImageIntegrationStorage() db.ImageIntegrationStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.ImageIntegrationStorage)
}

// GetLogsStorage provide storage functionality for logs sent to prevent central.
func GetLogsStorage() db.LogsStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.LogsStorage)
}

// GetMultiplierStorage provides the storage functionality for risk scoring multipliers
func GetMultiplierStorage() db.MultiplierStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.MultiplierStorage)
}

// GetNotifierStorage provide storage functionality for notifiers
func GetNotifierStorage() db.NotifierStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.NotifierStorage)
}

// GetServiceIdentityStorage provides storage functionality for service identities.
func GetServiceIdentityStorage() db.ServiceIdentityStorage {
	mutex.RLock()
	defer mutex.RUnlock()
	return registry.inmem.(db.ServiceIdentityStorage)
}
