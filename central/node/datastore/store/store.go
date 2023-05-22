package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for nodes.
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.Node, bool, error)
	// GetNodeMetadata and GetManyNodeMetadata returns the node without scan/component data.
	GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error)
	GetManyNodeMetadata(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error)

	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, node *storage.Node) error
	Delete(ctx context.Context, id string) error
}
