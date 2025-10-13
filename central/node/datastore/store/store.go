package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for nodes.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.Node, bool, error)
	// GetNodeMetadata and GetManyNodeMetadata returns the node without scan/component data.
	GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error)
	GetManyNodeMetadata(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(node *storage.Node) error) error

	Exists(ctx context.Context, id string) (bool, error)

	Upsert(ctx context.Context, node *storage.Node) error
	Delete(ctx context.Context, id string) error
}
