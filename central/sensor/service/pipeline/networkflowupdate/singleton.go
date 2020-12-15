package networkflowupdate

import (
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pi pipeline.FragmentFactory
)

func initialize() {
	pi = NewFactory(nfDS.Singleton(), networkBaselineManager.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.FragmentFactory {
	once.Do(initialize)
	return pi
}
