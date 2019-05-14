package networkflowupdate

import (
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pi pipeline.FragmentFactory
)

func initialize() {
	pi = NewFactory(nfDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.FragmentFactory {
	once.Do(initialize)
	return pi
}
