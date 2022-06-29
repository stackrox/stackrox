package bolt

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for undo records.
type Store interface {
	Get(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	Upsert(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
	Walk(ctx context.Context, fn func(np *storage.NetworkPolicyApplicationUndoRecord) error) error
}
