package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for CVEs.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NodeCVE, []int, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.NodeCVE) error) error

	UpsertMany(ctx context.Context, cves []*storage.NodeCVE) error
	PruneMany(ctx context.Context, ids []string) error
}
