package networkflowupdate

import (
	//"fmt"

	"github.com/gogo/protobuf/types"
	protobuf "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowStoreUpdaterImpl struct {
	flowStore store.FlowStore
	isFirst   bool
}

// update updates the FlowStore with the given network flow updates.
func (s *flowStoreUpdaterImpl) update(newFlows []*storage.NetworkFlow, updateTS *protobuf.Timestamp) error {
	updatedFlows := make(map[networkFlowProperties]timestamp.MicroTS, len(newFlows))

	// Add existing untermintated flows from the store if this is the first run.
	if s.isFirst {
		if err := s.addExistingNonTerminatedFlows(updatedFlows); err != nil {
			return err
		}
		s.isFirst = false
	}
	addNewFlows(updatedFlows, newFlows, updateTS)

	return s.flowStore.UpsertFlows(convertToFlows(updatedFlows), timestamp.Now())
}

func (s *flowStoreUpdaterImpl) addExistingNonTerminatedFlows(updatedFlows map[networkFlowProperties]timestamp.MicroTS) error {
	existingFlows, lastUpdateTS, err := s.flowStore.GetAllFlows(nil)
	if err != nil {
		return err
	}

	closeTS := timestamp.FromProtobuf(&lastUpdateTS)
	if closeTS == 0 {
		closeTS = timestamp.Now()
	}

	// Add non-terminated flows with the latest time in the store.
	// This will terminate the flow if it is not present in the incoming newFlows.
	for _, flow := range existingFlows {
		if flow.GetLastSeenTimestamp() == nil {
			updatedFlows[fromProto(flow.GetProps())] = closeTS
		}
	}
	return nil
}

func addNewFlows(updatedFlows map[networkFlowProperties]timestamp.MicroTS, newFlows []*storage.NetworkFlow, updateTS *protobuf.Timestamp) {
	tsOffset := timestamp.Now() - timestamp.FromProtobuf(updateTS)
	for _, newFlow := range newFlows {
		t := timestamp.FromProtobuf(newFlow.LastSeenTimestamp)
		if newFlow.LastSeenTimestamp != nil {
			t = t + tsOffset
		}
		updatedFlows[fromProto(newFlow.GetProps())] = t
	}
}

func convertToFlows(updatedFlows map[networkFlowProperties]timestamp.MicroTS) []*storage.NetworkFlow {
	flowsToBeUpserted := make([]*storage.NetworkFlow, 0, len(updatedFlows))
	for props, ts := range updatedFlows {
		toBeUpserted := &storage.NetworkFlow{
			Props:             props.toProto(),
			LastSeenTimestamp: convertTS(ts),
		}
		flowsToBeUpserted = append(flowsToBeUpserted, toBeUpserted)
	}
	return flowsToBeUpserted
}

func convertTS(ts timestamp.MicroTS) *types.Timestamp {
	if ts == 0 {
		return nil
	}
	return ts.GogoProtobuf()
}

// Helper class.
////////////////

type networkFlowProperties struct {
	srcEntity networkgraph.Entity
	dstEntity networkgraph.Entity
	dstPort   uint32
	protocol  storage.L4Protocol
}

func fromProto(protoProps *storage.NetworkFlowProperties) networkFlowProperties {
	return networkFlowProperties{
		srcEntity: networkgraph.EntityFromProto(protoProps.SrcEntity),
		dstEntity: networkgraph.EntityFromProto(protoProps.DstEntity),
		dstPort:   protoProps.DstPort,
		protocol:  protoProps.L4Protocol,
	}
}

func (n *networkFlowProperties) toProto() *storage.NetworkFlowProperties {
	return &storage.NetworkFlowProperties{
		SrcEntity:  n.srcEntity.ToProto(),
		DstEntity:  n.dstEntity.ToProto(),
		DstPort:    n.dstPort,
		L4Protocol: n.protocol,
	}
}
