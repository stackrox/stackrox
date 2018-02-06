package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("inmem")
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	db.AuthProviderStorage
	db.BenchmarkScansStorage
	db.BenchmarkScheduleStorage
	db.BenchmarkStorage
	db.ClusterStorage
	db.ImageStorage
	db.NotifierStorage
	db.ServiceIdentityStorage

	*alertStore
	*benchmarkTriggerStore
	*deploymentStore
	*policyStore
	*registryStore
	*scannerStore

	persistent db.Storage
}

// New creates a new InMemoryStore
func New(persistentStorage db.Storage) *InMemoryStore {
	return &InMemoryStore{
		AuthProviderStorage:      persistentStorage,
		BenchmarkScansStorage:    persistentStorage,
		BenchmarkScheduleStorage: persistentStorage,
		BenchmarkStorage:         persistentStorage,
		ClusterStorage:           persistentStorage,
		ImageStorage:             persistentStorage,
		NotifierStorage:          persistentStorage,
		ServiceIdentityStorage:   persistentStorage,

		persistent:            persistentStorage,
		alertStore:            newAlertStore(persistentStorage),
		benchmarkTriggerStore: newBenchmarkTriggerStore(persistentStorage),
		deploymentStore:       newDeploymentStore(persistentStorage),
		policyStore:           newPolicyStore(persistentStorage),
		registryStore:         newRegistryStore(persistentStorage),
		scannerStore:          newScannerStore(persistentStorage),
	}
}

// Close closes the persistent database
func (s *InMemoryStore) Close() {
	s.persistent.Close()
}
