// This file was originally generated with
// //go:generate cp ../../../../central/node/datastore/internal/store/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for nodes.
type Store interface {
	GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, node *storage.Node) error
}
