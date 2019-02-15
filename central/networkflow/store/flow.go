package store

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// FlowStore stores all of the flows for a single cluster.
//go:generate mockgen-wrapper FlowStore
type FlowStore interface {
	GetAllFlows(since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)
	GetFlow(props *storage.NetworkFlowProperties) (*storage.NetworkFlow, error)

	UpsertFlows(flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlow(props *storage.NetworkFlowProperties) error
}
