package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for nodes.
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.Node, bool, error)
	// GetNodeMetadata gets the node without scan/component data.
	GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, node *storage.Node) error
	Delete(ctx context.Context, id string) error
}
