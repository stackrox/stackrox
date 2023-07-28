package pings

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns the pipeline for ping messages.
func GetPipeline() pipeline.Fragment {
	return newPingPipeline()
}

func newPingPipeline() pipeline.Fragment {
	return &pipelineImpl{}
}

type pipelineImpl struct{}

func (s *pipelineImpl) OnFinish(_ string) {}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetPing() != nil
}

func (s *pipelineImpl) Run(ctx context.Context, clusterID string, _ *central.MsgFromSensor, injector common.MessageInjector) error {
	conn := connection.FromContext(ctx)

	if !conn.HasCapability(centralsensor.PingCap) {
		log.Warnf(`Cluster %q does not have the capability to receive pings.
It might have trouble with Central connection timeouts.`, clusterID)
		return nil
	}

	log.Debugf("Received ping from Cluster %q, responding with pong message.", clusterID)
	if err := injector.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_Pong{Pong: &central.CentralPong{}}}); err != nil {
		log.Warnf("Failed to answer ping message for Cluster %q: %v", clusterID, err)
	}
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
