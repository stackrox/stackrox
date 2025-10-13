package all

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/safe"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.ClusterPipeline = (*pipelineImpl)(nil)
)

// NewClusterPipeline returns a new instance of a ClusterPipeline that handles all event types.
func NewClusterPipeline(clusterID string, deduper hashManager.Deduper, fragments ...pipeline.Fragment) pipeline.ClusterPipeline {
	return &pipelineImpl{
		deduper:   deduper,
		fragments: fragments,
		clusterID: clusterID,
	}
}

type pipelineImpl struct {
	deduper   hashManager.Deduper
	clusterID string
	fragments []pipeline.Fragment
}

// Capabilities will return all capabilities of the ClusterPipeline.
func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	caps := make([]centralsensor.CentralCapability, 0, len(s.fragments))
	for _, fragment := range s.fragments {
		caps = append(caps, fragment.Capabilities()...)
	}
	return caps
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
	for _, fragment := range s.fragments {
		if fragment.Match(msg) {
			matchCount++

			var err error
			panicErr := safe.Run(func() {
				err = fragment.Run(ctx, s.clusterID, msg, injector)
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return errors.Wrap(err, "processing message from sensor")
			}
			if panicErr != nil {
				metrics.IncrementPipelinePanics(msg)
				return errors.Wrap(panicErr, "panic in pipeline execution")
			}
		}
	}
	if matchCount == 0 {
		return fmt.Errorf("no pipeline present to process message: %s", protocompat.MarshalTextString(msg))
	}
	s.deduper.MarkSuccessful(msg)
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {
	for _, fragment := range s.fragments {
		fragment.OnFinish(clusterID)
	}
}
