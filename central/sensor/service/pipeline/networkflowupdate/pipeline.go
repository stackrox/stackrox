package networkflowupdate

import (
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusterID string, storeUpdater flowStoreUpdater) pipeline.Fragment {
	return &pipelineImpl{
		clusterID:    clusterID,
		storeUpdater: storeUpdater,
	}
}

type pipelineImpl struct {
	clusterID    string
	storeUpdater flowStoreUpdater
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetNetworkFlowUpdate() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, _ pipeline.MsgInjector) (err error) {
	update := msg.GetNetworkFlowUpdate()

	if len(update.Updated) == 0 {
		return status.Errorf(codes.Internal, "received empty updated flows")
	}

	defer countMetrics.IncrementTotalNetworkFlowsReceivedCounter(s.clusterID, len(update.Updated))
	if err = s.storeUpdater.update(update.Updated, update.Time); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}
