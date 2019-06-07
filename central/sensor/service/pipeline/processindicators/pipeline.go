package processindicators

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(lifecycle.SingletonManager())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(manager lifecycle.Manager) pipeline.Fragment {
	return &pipelineImpl{
		manager: manager,
	}
}

type pipelineImpl struct {
	manager lifecycle.Manager
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetProcessIndicator() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ProcessIndicator)

	event := msg.GetEvent()
	switch event.GetAction() {
	case central.ResourceAction_CREATE_RESOURCE:
		indicator := event.GetProcessIndicator()
		indicator.ClusterId = clusterID
		return s.process(indicator, injector)
	default:
		return fmt.Errorf("action %q for process indicator is not supported", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *storage.ProcessIndicator, injector common.MessageInjector) error {
	return s.manager.IndicatorAdded(indicator, injector)
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
