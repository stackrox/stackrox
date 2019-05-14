package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// FlowDataStore stores all of the flows for a single cluster.
//go:generate mockgen-wrapper FlowDataStore
type FlowDataStore interface {
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)
	GetFlow(ctx context.Context, props *storage.NetworkFlowProperties) (*storage.NetworkFlow, error)

	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error

	RemoveFlowsForDeployment(ctx context.Context, id string) error
}
