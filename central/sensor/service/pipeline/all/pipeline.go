package all

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NewClusterPipeline returns a new instance of a ClusterPipeline that handles all event types.
func NewClusterPipeline(clusterID string, fragments ...pipeline.Fragment) pipeline.ClusterPipeline {
	return &pipelineImpl{
		fragments: fragments,
		clusterID: clusterID,
	}
}

type pipelineImpl struct {
	clusterID string
	fragments []pipeline.Fragment
}

// Reconcile passes through the reconciliation store to all the fragments and allows them to handle their reconciliation
// This will only happen once per cluster because the pipeline is generated every time the streamer connects
func (s *pipelineImpl) Reconcile(ctx context.Context, reconciliationStore *reconciliation.StoreMap) error {
	log.Info("Received Synced message from Sensor. Determining if there is any reconciliation to be done")
	errList := errorhelpers.NewErrorList("Reconciling state")
	for _, fragment := range s.fragments {
		errList.AddError(fragment.Reconcile(ctx, s.clusterID, reconciliationStore))
	}
	return errList.ToError()
}

// Run looks for one fragment (and only one) that matches the input message and runs that fragment on the message and injector.
func (s *pipelineImpl) Run(ctx context.Context, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	metrics.SetResourceProcessingDuration(msg.GetEvent())
	defer metrics.SetSensorEventRunDuration(time.Now(), common.GetMessageType(msg), msg.GetEvent().GetAction().String())

	var matchCount int
	errorList := errorhelpers.NewErrorList("error processing message from sensor")
	for _, fragment := range s.fragments {
		if fragment.Match(msg) {
			matchCount++
			errorList.AddError(fragment.Run(ctx, s.clusterID, msg, injector))
		}
	}
	if matchCount == 0 {
		return fmt.Errorf("no pipeline present to process message: %s", proto.MarshalTextString(msg))
	}
	return errorList.ToError()
}

func (s *pipelineImpl) OnFinish(clusterID string) {
	for _, fragment := range s.fragments {
		fragment.OnFinish(clusterID)
	}
}
