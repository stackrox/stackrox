package undodeploymentstore

import (
	"github.com/stackrox/rox/generated/storage"
)

// UndoDeploymentStore provides storage functionality for network baselines.
//go:generate mockgen-wrapper
type UndoDeploymentStore interface {
	Get(deploymentID string) (*storage.NetworkPolicyApplicationUndoDeploymentRecord, bool, error)
	Upsert(undoRecord *storage.NetworkPolicyApplicationUndoDeploymentRecord) error
}
