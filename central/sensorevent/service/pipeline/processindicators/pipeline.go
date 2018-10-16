package processindicators

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators processDataStore.DataStore, deployments datastore.DataStore, manager lifecycle.Manager) pipeline.Pipeline {
	return &pipelineImpl{
		indicators:  indicators,
		manager:     manager,
		deployments: deployments,
	}
}

type pipelineImpl struct {
	indicators  processDataStore.DataStore
	deployments datastore.DataStore
	manager     lifecycle.Manager
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent, injector pipeline.EnforcementInjector) error {
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return s.process(event.GetProcessIndicator(), injector)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *v1.ProcessIndicator, injector pipeline.EnforcementInjector) error {
	err := s.indicators.AddProcessIndicator(indicator)
	if err != nil {
		return err
	}
	return s.manager.IndicatorAdded(indicator, injector)
}
