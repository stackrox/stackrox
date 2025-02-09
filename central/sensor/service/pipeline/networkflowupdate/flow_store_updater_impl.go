package networkflowupdate

import (
	"context"
	"time"

	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	flowDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowPersisterImpl struct {
	seenBaselineRelevantFlows map[networkgraph.NetworkConnIndicator]struct{}

	baselines       networkBaselineManager.Manager
	flowStore       flowDataStore.FlowDataStore
	firstUpdateSeen bool
	entityStore     entityDataStore.EntityDataStore
	clusterID       string
}

// update updates the FlowStore with the given network flow updates.
func (s *flowPersisterImpl) update(ctx context.Context, newFlows []*storage.NetworkFlow, updateTS *time.Time) error {
	if features.ExternalIPs.Enabled() {
		// Sensor may have forwarded unknown NetworkEntities that we want to learn
		for _, newFlow := range newFlows {
			err := s.fixupExternalNetworkEntityIdIfDiscovered(ctx, newFlow.GetProps().DstEntity)
			if err != nil {
				return err
			}
			err = s.updateExternalNetworkEntityIfDiscovered(ctx, newFlow.GetProps().DstEntity)
			if err != nil {
				return err
			}
			err = s.fixupExternalNetworkEntityIdIfDiscovered(ctx, newFlow.GetProps().SrcEntity)
			if err != nil {
				return err
			}
			err = s.updateExternalNetworkEntityIfDiscovered(ctx, newFlow.GetProps().SrcEntity)
			if err != nil {
				return err
			}
		}
	} else {
		// We are not storing the discovered entities. Let net-flows point to INTERNET instead.
		internetEntity := networkgraph.InternetEntity().ToProto()
		for _, newFlow := range newFlows {
			if newFlow.GetProps().GetSrcEntity().GetExternalSource().GetDiscovered() {
				newFlow.GetProps().SrcEntity = internetEntity
			}

			if newFlow.GetProps().GetDstEntity().GetExternalSource().GetDiscovered() {
				newFlow.GetProps().DstEntity = internetEntity
			}
		}
	}

	now := timestamp.Now()
	var updateMicroTS timestamp.MicroTS
	if updateTS != nil {
		updateMicroTS = timestamp.FromGoTime(*updateTS)
	}

	flowsByIndicator := getFlowsByIndicator(newFlows, updateMicroTS, now)
	if err := s.baselines.ProcessFlowUpdate(flowsByIndicator); err != nil {
		return err
	}

	// Add existing unterminated flows from the store if this is the first run this time round.
	if !s.firstUpdateSeen {
		if err := s.markExistingFlowsAsTerminatedIfNotSeen(ctx, flowsByIndicator); err != nil {
			return err
		}
		s.firstUpdateSeen = true
	}

	return s.flowStore.UpsertFlows(ctx, convertToFlows(flowsByIndicator), now)
}

func (s *flowPersisterImpl) markExistingFlowsAsTerminatedIfNotSeen(ctx context.Context, currentFlows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) error {
	existingFlows, lastUpdateTS, err := s.flowStore.GetAllFlows(ctx, nil)
	if err != nil {
		return err
	}

	var closeTS timestamp.MicroTS
	if lastUpdateTS != nil {
		closeTS = timestamp.FromGoTime(*lastUpdateTS)
	}
	if closeTS == 0 {
		closeTS = timestamp.Now()
	}

	// If there are flows in the store that are not terminated, and which are NOT present in the currentFlows,
	// then we need to mark them as terminated, with a timestamp of closeTS.
	for _, flow := range existingFlows {
		// An empty last seen timestamp means the flow had not been terminated
		// the last time we wrote it.
		if flow.GetLastSeenTimestamp() == nil {
			indicator := networkgraph.GetNetworkConnIndicator(flow)
			if _, stillExists := currentFlows[indicator]; !stillExists {
				currentFlows[indicator] = closeTS
			}
		}
	}
	return nil
}

func getFlowsByIndicator(newFlows []*storage.NetworkFlow, updateTS, now timestamp.MicroTS) map[networkgraph.NetworkConnIndicator]timestamp.MicroTS {
	out := make(map[networkgraph.NetworkConnIndicator]timestamp.MicroTS, len(newFlows))
	tsOffset := now - updateTS
	for _, newFlow := range newFlows {
		t := timestamp.FromProtobuf(newFlow.LastSeenTimestamp)
		if newFlow.LastSeenTimestamp != nil {
			t = t + tsOffset
		}
		out[networkgraph.GetNetworkConnIndicator(newFlow)] = t
	}
	return out
}

func convertToFlows(updatedFlows map[networkgraph.NetworkConnIndicator]timestamp.MicroTS) []*storage.NetworkFlow {
	flowsToBeUpserted := make([]*storage.NetworkFlow, 0, len(updatedFlows))
	for indicator, ts := range updatedFlows {
		toBeUpserted := &storage.NetworkFlow{
			Props: indicator.ToNetworkFlowPropertiesProto(),
		}
		if ts != 0 {
			toBeUpserted.LastSeenTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
		}
		flowsToBeUpserted = append(flowsToBeUpserted, toBeUpserted)
	}
	return flowsToBeUpserted
}

func (s *flowPersisterImpl) updateExternalNetworkEntityIfDiscovered(ctx context.Context, entityInfo *storage.NetworkEntityInfo) error {
	if !entityInfo.GetExternalSource().GetDiscovered() {
		return nil
	}

	// Discovered entities are stored
	entity := &storage.NetworkEntity{
		Info: entityInfo,
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: s.clusterID,
		},
	}

	return s.entityStore.UpdateExternalNetworkEntity(ctx, entity, true)
}

// Sensor cannot put the correct clusterId in discovered entities, but we have the necessary information.
// In the particular case of discovered, we replace the entity ID with a scoped ID matching the cluster
// we received this entity from.
func (s *flowPersisterImpl) fixupExternalNetworkEntityIdIfDiscovered(ctx context.Context, entityInfo *storage.NetworkEntityInfo) error {
	if !entityInfo.GetExternalSource().GetDiscovered() {
		return nil
	}

	id, err := externalsrcs.NewClusterScopedID(s.clusterID, entityInfo.GetExternalSource().GetCidr())

	if err == nil {
		entityInfo.Id = id.String()
	}

	return err
}
