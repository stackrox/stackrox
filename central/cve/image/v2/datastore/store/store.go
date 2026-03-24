package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for normalized CVEs.
//
//go:generate mockgen-wrapper
type Store interface {
	// Standard generated CRUD methods from pg-table-bindings-wrapper.

	Upsert(ctx context.Context, obj *storage.NormalizedCVE) error
	UpsertMany(ctx context.Context, objs []*storage.NormalizedCVE) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.NormalizedCVE, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NormalizedCVE, []int, error)
	GetIDs(ctx context.Context) ([]string, error)
	Walk(ctx context.Context, fn func(*storage.NormalizedCVE) error) error
}
