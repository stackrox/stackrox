package store

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore
	Walk(ctx context.Context, fn func(clusterID string, ts *types.Timestamp, allFlows []*storage.NetworkFlow) error) error
}

// FlowStore stores all the flows for a single cluster.
type FlowStore interface {
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
}
