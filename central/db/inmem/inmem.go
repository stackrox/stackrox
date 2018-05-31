package inmem

import (
	"net/http"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	db.AlertStorage
	db.AuthProviderStorage
	db.BenchmarkScansStorage
	db.BenchmarkScheduleStorage
	db.BenchmarkStorage
	db.ClusterStorage
	db.DeploymentStorage
	db.ImageStorage
	db.LogsStorage
	db.MultiplierStorage
	db.NotifierStorage
	db.PolicyStorage
	db.ServiceIdentityStorage

	*benchmarkTriggerStore
	*imageIntegrationStore

	persistent db.Storage
}

// New creates a new InMemoryStore
func New(persistentStorage db.Storage) *InMemoryStore {
	return &InMemoryStore{
		AlertStorage:             persistentStorage,
		AuthProviderStorage:      persistentStorage,
		BenchmarkScansStorage:    persistentStorage,
		BenchmarkScheduleStorage: persistentStorage,
		BenchmarkStorage:         persistentStorage,
		ClusterStorage:           persistentStorage,
		DeploymentStorage:        persistentStorage,
		ImageStorage:             persistentStorage,
		LogsStorage:              persistentStorage,
		MultiplierStorage:        persistentStorage,
		NotifierStorage:          persistentStorage,
		PolicyStorage:            persistentStorage,
		ServiceIdentityStorage:   persistentStorage,

		persistent:            persistentStorage,
		benchmarkTriggerStore: newBenchmarkTriggerStore(persistentStorage),
		imageIntegrationStore: newImageIntegrationStore(persistentStorage),
	}
}

// Close closes the persistent database
func (s *InMemoryStore) Close() {
	s.persistent.Close()
}

// BackupHandler returns the persistent database's BackupHandler
func (s *InMemoryStore) BackupHandler() http.Handler {
	return s.persistent.BackupHandler()
}

// ExportHandler returns the persistent database's ExportHandler
func (s *InMemoryStore) ExportHandler() http.Handler {
	return s.persistent.ExportHandler()
}
