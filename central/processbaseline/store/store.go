package store

import (
	"context"

	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process baselines.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.ProcessBaseline, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ProcessBaseline, []int, error)
	Walk(ctx context.Context, fn func(baseline *storage.ProcessBaseline) error) error

	Upsert(ctx context.Context, baseline *storage.ProcessBaseline) error
	UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error

	Delete(ctx context.Context, id string) error
}
