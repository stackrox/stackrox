package store

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/timestamp"
)

const networkFlowBucket = "networkFlows"

// FlowStore stores all of the flows for a single cluster.
//go:generate mockery -name=FlowStore
type FlowStore interface {
	GetAllFlows() ([]*v1.NetworkFlow, types.Timestamp, error)
	GetFlow(props *v1.NetworkFlowProperties) (*v1.NetworkFlow, error)

	UpsertFlows(flows []*v1.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlow(props *v1.NetworkFlowProperties) error
}
