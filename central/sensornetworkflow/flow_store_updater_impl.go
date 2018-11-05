package sensornetworkflow

import (
	//"fmt"

	"github.com/gogo/protobuf/types"
	protobuf "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowStoreUpdaterImpl struct {
	flowStore store.FlowStore
	isFirst   bool
}

// update updates the FlowStore with the given network flow updates.
func (s *flowStoreUpdaterImpl) update(newFlows []*v1.NetworkFlow, updateTS *protobuf.Timestamp) error {
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
	existingFlows, lastUpdateTS, err := s.flowStore.GetAllFlows()
	if err != nil {
		return err
	}

	// Add non-terminated flows with the latest time in the store.
	// This will terminate the flow if it is not present in the incoming newFlows.
	for _, flow := range existingFlows {
		if flow.GetLastSeenTimestamp() == nil {
			updatedFlows[fromProto(flow.GetProps())] = timestamp.FromProtobuf(&lastUpdateTS)
		}
	}
	return nil
}

func addNewFlows(updatedFlows map[networkFlowProperties]timestamp.MicroTS, newFlows []*v1.NetworkFlow, updateTS *protobuf.Timestamp) {
	tsOffset := timestamp.Now() - timestamp.FromProtobuf(updateTS)
	for _, newFlow := range newFlows {
		t := timestamp.FromProtobuf(newFlow.LastSeenTimestamp)
		if newFlow.LastSeenTimestamp != nil {
			t = t + tsOffset
		}
		updatedFlows[fromProto(newFlow.GetProps())] = t
	}
}

func convertToFlows(updatedFlows map[networkFlowProperties]timestamp.MicroTS) []*v1.NetworkFlow {
	flowsToBeUpserted := make([]*v1.NetworkFlow, 0, len(updatedFlows))
	for props, ts := range updatedFlows {
		toBeUpserted := &v1.NetworkFlow{
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
	srcDeploymentID string
	dstDeploymentID string
	dstPort         uint32
	protocol        v1.L4Protocol
}

func fromProto(protoProps *v1.NetworkFlowProperties) networkFlowProperties {
	return networkFlowProperties{
		srcDeploymentID: protoProps.SrcDeploymentId,
		dstDeploymentID: protoProps.DstDeploymentId,
		dstPort:         protoProps.DstPort,
		protocol:        protoProps.L4Protocol,
	}
}

func (n *networkFlowProperties) toProto() *v1.NetworkFlowProperties {
	return &v1.NetworkFlowProperties{
		SrcDeploymentId: n.srcDeploymentID,
		DstDeploymentId: n.dstDeploymentID,
		DstPort:         n.dstPort,
		L4Protocol:      n.protocol,
	}
}
