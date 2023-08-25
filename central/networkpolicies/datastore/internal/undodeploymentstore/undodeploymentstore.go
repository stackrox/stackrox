package undodeploymentstore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// UndoDeploymentStore provides storage functionality for network baselines.
//
//go:generate mockgen-wrapper
type UndoDeploymentStore interface {
	Get(ctx context.Context, deploymentID string) (*storage.NetworkPolicyApplicationUndoDeploymentRecord, bool, error)
	Upsert(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoDeploymentRecord) error
}
