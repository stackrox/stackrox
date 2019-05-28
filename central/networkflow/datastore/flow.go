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
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlowsForDeployment(ctx context.Context, id string) error
}
