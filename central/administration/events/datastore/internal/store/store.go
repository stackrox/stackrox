package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the events data layer.
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, id string) (*storage.AdministrationEvent, bool, error)
	// Deprecated: use GetByQueryFn instead
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.AdministrationEvent) error) error
	UpsertMany(ctx context.Context, objs []*storage.AdministrationEvent) error
	DeleteMany(ctx context.Context, identifiers []string) error
	GetMany(ctx context.Context, identifiers []string) ([]*storage.AdministrationEvent, []int, error)
}
