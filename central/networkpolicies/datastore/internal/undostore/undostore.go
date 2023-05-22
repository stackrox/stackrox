package undostore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// UndoStore provides storage functionality for undo records.
//
//go:generate mockgen-wrapper
type UndoStore interface {
	Get(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	Upsert(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
}
