package events

import (
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// NodeRoleChangeEvent is published when the set of MachineConfigPools in the cluster changes.
// It carries no payload — consumers should query the node-role store for the current state.
type NodeRoleChangeEvent struct{}

func (e *NodeRoleChangeEvent) Topic() pubsub.Topic { return pubsub.NodeRoleChangeTopic }
func (e *NodeRoleChangeEvent) Lane() pubsub.LaneID { return pubsub.NodeRoleChangeLane }
