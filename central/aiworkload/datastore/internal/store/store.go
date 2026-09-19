package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AIWorkloadStore provides storage functionality for AI workloads.
//
//go:generate mockgen-wrapper
type AIWorkloadStore interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.AIWorkload, bool, error)
	Walk(ctx context.Context, fn func(workload *storage.AIWorkload) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(workload *storage.AIWorkload) error) error

	DeleteMany(ctx context.Context, identifiers []string) error
	UpsertMany(ctx context.Context, objs []*storage.AIWorkload) error
}
