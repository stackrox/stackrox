package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

// FlowDataStore stores all of the flows for a single cluster.
//
//go:generate mockgen-wrapper
type FlowDataStore interface {
	GetAllFlows(ctx context.Context, since *time.Time) ([]*storage.NetworkFlow, *time.Time, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *time.Time) ([]*storage.NetworkFlow, *time.Time, error)
	// GetFlowsForDeployment returns all flows referencing a specific deployment id
	GetFlowsForDeployment(ctx context.Context, deploymentID string, adjustForGraph bool) ([]*storage.NetworkFlow, error)
	// GetExternalFlowsForDeployment returns all external flows referencing a specific deployment id
	GetExternalFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error)

	// UpsertFlows upserts the given flows to the store. The flows slice might be modified by this function, so if you
	// need to use it afterwards, create a copy.
	// Returns the flows actually upserted.
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) ([]*storage.NetworkFlow, error)
	RemoveFlowsForDeployment(ctx context.Context, id string) error
	RemoveStaleFlows(ctx context.Context) error
	// RemoveOrphanedFlows - remove flows that have been orphaned by deployments
	RemoveOrphanedFlows(ctx context.Context, orphanWindow *time.Time) error
}
