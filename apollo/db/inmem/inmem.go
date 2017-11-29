package inmem

import (
	"fmt"
	"reflect"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log        = logging.New("inmem")
	loaderType = reflect.TypeOf((*loader)(nil)).Elem()
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	*alertStore
	*benchmarkStore
	*deploymentStore
	*imagePolicyStore
	*imageStore
	*registryStore
	*scannerStore

	persistent db.Storage
}

// New creates a new InMemoryStore
func New(persistentStorage db.Storage) *InMemoryStore {
	return &InMemoryStore{
		persistent:       persistentStorage,
		alertStore:       newAlertStore(persistentStorage),
		benchmarkStore:   newBenchmarkStore(persistentStorage),
		deploymentStore:  newDeploymentStore(persistentStorage),
		imagePolicyStore: newImagePolicyStore(persistentStorage),
		imageStore:       newImageStore(persistentStorage),
		registryStore:    newRegistryStore(persistentStorage),
		scannerStore:     newScannerStore(persistentStorage),
	}
}

// Load initializes the in-memory database from the persistent database
func (s *InMemoryStore) Load() error {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		vField := v.Field(i)
		tField := t.Field(i)

		if tField.Type.Implements(loaderType) {
			if err := vField.Interface().(loader).loadFromPersistent(); err != nil {
				return fmt.Errorf("unable to load data from persistent storage for %s", tField.Name)
			}
		} else {
			log.Infof("Field %v does not support loading from persistence", tField.Name)
		}
	}
	return nil
}

// Close closes the persistent database
func (s *InMemoryStore) Close() {
	s.persistent.Close()
}

type loader interface {
	loadFromPersistent() error
}
