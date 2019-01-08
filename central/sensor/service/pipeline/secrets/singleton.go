package secrets

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(), secretDataStore.Singleton())

}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return pi
}
