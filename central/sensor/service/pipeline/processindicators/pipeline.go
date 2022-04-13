package processindicators

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/process/id"
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

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, _ *reconciliation.StoreMap) error {
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

		// Build indicator from exec filepath, process, and args
		// This allows for a consistent ID to be inserted into the DB
		id.SetIndicatorID(indicator)

		return s.process(indicator)
	default:
		return errors.Errorf("action %q for process indicator is not supported", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *storage.ProcessIndicator) error {
	return s.manager.IndicatorAdded(indicator)
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
