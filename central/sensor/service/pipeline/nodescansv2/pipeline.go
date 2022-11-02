package nodescansv2

import (
	"context"

	"github.com/pkg/errors"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline()
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline() pipeline.Fragment {
	return &pipelineImpl{}
}

type pipelineImpl struct {
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNodeScanV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NodeScanV2)

	event := msg.GetEvent()
	nodeScan := event.GetNodeScanV2()
	if nodeScan == nil {
		return errors.Errorf("unexpected resource type %T for node scan v2", event.GetResource())
	}

	// TODO(ROX-12240, ROX-13053): Do something meaningful with the nodeScan
	log.Infof("Central received NodeScanV2: %+v", nodeScan)

	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
