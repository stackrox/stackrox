package processindicators

import (
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators datastore.DataStore) pipeline.Pipeline {
	return &pipelineImpl{
		indicators: indicators,
	}
}

type pipelineImpl struct {
	indicators datastore.DataStore
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEventResponse, error) {
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return nil, s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return nil, s.process(event.GetProcessIndicator())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *v1.ProcessIndicator) error {
	return s.indicators.AddProcessIndicator(indicator)
}
