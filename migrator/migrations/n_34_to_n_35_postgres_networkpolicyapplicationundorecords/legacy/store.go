// This file was originally generated with
// //go:generate cp ../../../../central/networkpolicies/datastore/internal/undostore/undostore.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for undo records.
type Store interface {
	Upsert(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
	Walk(ctx context.Context, fn func(np *storage.NetworkPolicyApplicationUndoRecord) error) error
}
