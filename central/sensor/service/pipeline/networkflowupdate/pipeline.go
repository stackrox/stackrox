package networkflowupdate

import (
	"context"
	"errors"
	"fmt"

	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusterID string, storeUpdater flowPersister) pipeline.Fragment {
	return &pipelineImpl{
		clusterID:    clusterID,
		storeUpdater: storeUpdater,
	}
}

type pipelineImpl struct {
	clusterID    string
	storeUpdater flowPersister
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetNetworkFlowUpdate() != nil
}

var seen = make(map[string]*storage.NetworkFlow)

func GetID(props *storage.NetworkFlowProperties) string {
	return fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(props.GetSrcEntity().GetType()), props.GetSrcEntity().GetId(), int32(props.GetDstEntity().GetType()), props.GetDstEntity().GetId(), props.GetDstPort(), int32(props.GetL4Protocol()))
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) (err error) {
	update := msg.GetNetworkFlowUpdate()

	if len(update.GetUpdated())+len(update.GetUpdatedEndpoints()) == 0 {
		return errors.New("received empty updated flows")
	}

	countMetrics.IncrementTotalNetworkFlowsReceivedCounter(s.clusterID, len(update.GetUpdated()))

	var allUpdatedFlows []*storage.NetworkFlow
	allUpdatedFlows = make([]*storage.NetworkFlow, 0, len(update.GetUpdated())+len(update.GetUpdatedEndpoints()))
	allUpdatedFlows = append(allUpdatedFlows, update.GetUpdated()...)
	allUpdatedFlows = append(allUpdatedFlows, endpointsToListenFlows(update.GetUpdatedEndpoints())...)
	countMetrics.IncrementTotalNetworkEndpointsReceivedCounter(s.clusterID, len(update.GetUpdatedEndpoints()))

	for _, flow := range allUpdatedFlows {
		id := GetID(flow.GetProps())
		_, ok := seen[id]
		if ok {
			log.Infof("Already seen flow: %s", flow)
		} else {
			seen[id] = flow
		}
	}
	if len(allUpdatedFlows) == 0 {
		return nil
	}

	if err = s.storeUpdater.update(ctx, allUpdatedFlows, update.Time); err != nil {
		return err
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}

func endpointsToListenFlows(endpoints []*storage.NetworkEndpoint) []*storage.NetworkFlow {
	listenFlows := make([]*storage.NetworkFlow, 0, len(endpoints))

	for _, ep := range endpoints {
		listenFlows = append(listenFlows, &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: ep.GetProps().GetEntity(),
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_LISTEN_ENDPOINT,
				},
				DstPort:    ep.GetProps().GetPort(),
				L4Protocol: ep.GetProps().GetL4Protocol(),
			},
			LastSeenTimestamp: ep.GetLastActiveTimestamp(),
		})
	}
	return listenFlows
}
