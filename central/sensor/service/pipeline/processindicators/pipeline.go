package processindicators

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators processDataStore.DataStore, deployments datastore.DataStore, manager lifecycle.Manager) pipeline.Fragment {
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

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetProcessIndicator() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, injector pipeline.MsgInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ProcessIndicator)

	event := msg.GetEvent()
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return s.process(event.GetProcessIndicator(), injector)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *storage.ProcessIndicator, injector pipeline.MsgInjector) error {
	return s.manager.IndicatorAdded(indicator, injector)
}

func (s *pipelineImpl) OnFinish() {}
