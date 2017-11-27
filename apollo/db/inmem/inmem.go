package inmem

import (
	"fmt"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("inmem")
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	images     map[string]*v1.Image
	imageMutex sync.Mutex

	imageRules      map[string]*v1.ImageRule
	imageRulesMutex sync.Mutex

	alerts     map[string]*v1.Alert
	alertMutex sync.Mutex

	registries    map[string]registryTypes.ImageRegistry
	registryMutex sync.Mutex

	scanners  map[string]scannerTypes.ImageScanner
	scanMutex sync.Mutex

	benchmarks     map[string]*v1.BenchmarkPayload
	benchmarkMutex sync.Mutex

	persistent db.Storage
}

// New creates a new InMemoryStore
func New(persistentStorage db.Storage) *InMemoryStore {
	return &InMemoryStore{
		images:     make(map[string]*v1.Image),
		imageRules: make(map[string]*v1.ImageRule),
		alerts:     make(map[string]*v1.Alert),
		registries: make(map[string]registryTypes.ImageRegistry),
		scanners:   make(map[string]scannerTypes.ImageScanner),
		benchmarks: make(map[string]*v1.BenchmarkPayload),
		persistent: persistentStorage,
	}
}

// Load initializes the in-memory database from the persistent database
func (i *InMemoryStore) Load() error {
	if err := i.loadImages(); err != nil {
		return fmt.Errorf("Errors loading images: %+v", err)
	}
	if err := i.loadImageRules(); err != nil {
		return fmt.Errorf("Errors loading image rules: %+v", err)
	}
	if err := i.loadAlerts(); err != nil {
		return fmt.Errorf("Errors loading alerts: %+v", err)
	}
	return nil
}

// Close closes the persistent database
func (i *InMemoryStore) Close() {
	i.persistent.Close()
}
