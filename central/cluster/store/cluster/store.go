package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for clusters.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.Cluster, bool, error)
	Upsert(ctx context.Context, obj *storage.Cluster) error
	Delete(ctx context.Context, id string) error
	GetMany(ctx context.Context, ids []string) ([]*storage.Cluster, []int, error)

	Walk(ctx context.Context, fn func(obj *storage.Cluster) error) error
}
