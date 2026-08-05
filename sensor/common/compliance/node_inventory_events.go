package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// NodeInventoryEvent carries a NodeInventory relayed from a compliance gRPC node.
type NodeInventoryEvent struct {
	Inventory *storage.NodeInventory
}

func (e *NodeInventoryEvent) Topic() pubsub.Topic {
	return pubsub.NodeInventoryTopic
}

// Lane returns NodeInventoryIntakeLane, shared with IndexReportWrapEvent so that both
// event types are delivered serially and preserve nodeInventoryHandlerImpl.archCache's
// single-goroutine safety guarantee.
func (e *NodeInventoryEvent) Lane() pubsub.LaneID {
	return pubsub.NodeInventoryIntakeLane
}

// IndexReportWrapEvent carries an IndexReportWrap relayed from a compliance gRPC node.
type IndexReportWrapEvent struct {
	Wrap *index.IndexReportWrap
}

func (e *IndexReportWrapEvent) Topic() pubsub.Topic {
	return pubsub.IndexReportWrapTopic
}

// Lane returns NodeInventoryIntakeLane, shared with NodeInventoryEvent so that both
// event types are delivered serially and preserve nodeInventoryHandlerImpl.archCache's
// single-goroutine safety guarantee.
func (e *IndexReportWrapEvent) Lane() pubsub.LaneID {
	return pubsub.NodeInventoryIntakeLane
}
