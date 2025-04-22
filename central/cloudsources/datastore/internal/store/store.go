package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store is the interface to the cloud sources data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Walk(ctx context.Context, fn func(obj *storage.CloudSource) error) error
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.CloudSource, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.CloudSource, error)
	Upsert(ctx context.Context, obj *storage.CloudSource) error
	Delete(ctx context.Context, id string) error
}
