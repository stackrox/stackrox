package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality.
//
//go:generate mockgen-wrapper
type Store interface {
	Upsert(ctx context.Context, obj *storage.RuntimeFilterData) error
	UpsertMany(ctx context.Context, objs []*storage.RuntimeFilterData) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)
	DeleteMany(ctx context.Context, identifiers []string) error

	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.RuntimeFilterData, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.RuntimeFilterData, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storage.RuntimeFilterData, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storage.RuntimeFilterData) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.RuntimeFilterData) error) error
}
