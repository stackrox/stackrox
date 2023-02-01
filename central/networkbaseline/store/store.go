package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for network baselines.
//
//go:generate mockgen-wrapper
type Store interface {
	Exists(ctx context.Context, id string) (bool, error)

	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.NetworkBaseline, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NetworkBaseline, []int, error)

	Upsert(ctx context.Context, baseline *storage.NetworkBaseline) error
	UpsertMany(ctx context.Context, baselines []*storage.NetworkBaseline) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	Walk(ctx context.Context, fn func(baseline *storage.NetworkBaseline) error) error
}
