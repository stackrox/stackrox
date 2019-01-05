package all

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/compliancereturn"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/deploymentevents"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/namespaces"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/networkpolicies"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/nodes"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/processindicators"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/providermetadata"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/secrets"
)

var (
	once sync.Once

	allPipeline pipeline.Pipeline
)

func initialize() {
	allPipeline = NewPipeline(deploymentevents.Singleton(),
		processindicators.Singleton(),
		networkpolicies.Singleton(),
		namespaces.Singleton(),
		secrets.Singleton(),
		nodes.Singleton(),
		providermetadata.Singleton(),
		compliancereturn.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return allPipeline
}
