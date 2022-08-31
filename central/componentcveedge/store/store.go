package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for component-cve edges.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.ComponentCVEEdge, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ComponentCVEEdge, []int, error)
}
