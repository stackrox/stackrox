package store

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(ctx context.Context, clusterID string) (FlowStore, error)

	Walk(ctx context.Context, fn func(clusterID string, ts types.Timestamp, allFlows []*storage.NetworkFlow) error) error
}
