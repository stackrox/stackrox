package processindicators

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
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
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error) {
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return nil, s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return s.process(event.GetProcessIndicator())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *v1.ProcessIndicator) (*v1.SensorEnforcement, error) {
	deployment, exists, err := s.deployments.GetDeployment(indicator.GetDeploymentId())
	if err != nil {
		return nil, fmt.Errorf("error retrieving deployment from indicator %s: %s", proto.MarshalTextString(indicator), err)
	}
	if !exists {
		return nil, fmt.Errorf("received indicator %s for non-existent deployment", proto.MarshalTextString(indicator))
	}

	inserted, err := s.indicators.AddProcessIndicator(indicator)
	if err != nil {
		return nil, err
	}
	// This short-circuits the processing for de-duplicated indicators.
	if !inserted {
		return nil, nil
	}
	return s.manager.IndicatorAdded(indicator, deployment)
}
