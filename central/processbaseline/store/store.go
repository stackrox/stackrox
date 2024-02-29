package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for process baselines.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.ProcessBaseline, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ProcessBaseline, []int, error)
	Walk(ctx context.Context, fn func(baseline *storage.ProcessBaseline) error) error

	Upsert(ctx context.Context, baseline *storage.ProcessBaseline) error
	UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error

	Delete(ctx context.Context, id string) error
}
