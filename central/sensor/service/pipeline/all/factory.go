package all

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/compliancereturn"
	"github.com/stackrox/rox/central/sensor/service/pipeline/deploymentevents"
	"github.com/stackrox/rox/central/sensor/service/pipeline/namespaces"
	"github.com/stackrox/rox/central/sensor/service/pipeline/networkflowupdate"
	"github.com/stackrox/rox/central/sensor/service/pipeline/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/rox/central/sensor/service/pipeline/processindicators"
	"github.com/stackrox/rox/central/sensor/service/pipeline/providermetadata"
	"github.com/stackrox/rox/central/sensor/service/pipeline/secrets"
)

// NewFactory returns a new instance of a Factory that produces a pipeline handling all message types.
func NewFactory() pipeline.Factory {
	return &factoryImpl{}
}

type factoryImpl struct{}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *factoryImpl) GetPipeline(clusterID string) (pipeline.Pipeline, error) {
	flowUpdateFragment, err := networkflowupdate.Singleton().GetFragment(clusterID)
	if err != nil {
		return nil, err
	}

	return NewPipeline(deploymentevents.Singleton(),
		processindicators.Singleton(),
		networkpolicies.Singleton(),
		namespaces.Singleton(),
		secrets.Singleton(),
		nodes.Singleton(),
		providermetadata.Singleton(),
		compliancereturn.Singleton(),
		flowUpdateFragment,
	), nil
}
