package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("inmem")
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	*alertStore
	*benchmarkScheduleStore
	*benchmarkResultStore
	*benchmarkStore
	*benchmarkTriggerStore
	*clusterStore
	*deploymentStore
	*policyStore
	*imageStore
	*notifierStore
	*registryStore
	*scannerStore

	persistent db.Storage
}

// New creates a new InMemoryStore
func New(persistentStorage db.Storage) *InMemoryStore {
	return &InMemoryStore{
		persistent:             persistentStorage,
		alertStore:             newAlertStore(persistentStorage),
		benchmarkScheduleStore: newBenchmarkScheduleStore(persistentStorage),
		benchmarkStore:         newBenchmarkStore(persistentStorage),
		benchmarkResultStore:   newBenchmarkResultsStore(persistentStorage),
		benchmarkTriggerStore:  newBenchmarkTriggerStore(persistentStorage),
		clusterStore:           newClusterStore(persistentStorage),
		deploymentStore:        newDeploymentStore(persistentStorage),
		policyStore:            newPolicyStore(persistentStorage),
		imageStore:             newImageStore(persistentStorage),
		notifierStore:          newNotifierStore(persistentStorage),
		registryStore:          newRegistryStore(persistentStorage),
		scannerStore:           newScannerStore(persistentStorage),
	}
}

// Close closes the persistent database
func (s *InMemoryStore) Close() {
	s.persistent.Close()
}
