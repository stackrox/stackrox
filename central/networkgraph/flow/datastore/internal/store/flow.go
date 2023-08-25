package store

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// FlowStore stores all of the flows for a single cluster.
//
//go:generate mockgen-wrapper
type FlowStore interface {
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	// GetFlowsForDeployment returns all flows referencing a specific deployment id
	GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error)

	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error

	RemoveFlowsForDeployment(ctx context.Context, id string) error

	// RemoveStaleFlows - remove stale duplicate network flows
	RemoveStaleFlows(ctx context.Context) error

	// RemoveOrphanedFlows - remove flows that have been orphaned by deployments
	RemoveOrphanedFlows(ctx context.Context, orphanWindow *time.Time) error
}
