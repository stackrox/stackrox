package networkflowupdate

import (
	networkBaselineManager "github.com/stackrox/stackrox/central/networkbaseline/manager"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/pkg/sync"
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
