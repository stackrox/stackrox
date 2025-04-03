package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides the interface to the underlying storage
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.NetworkPolicy, error)
	Upsert(ctx context.Context, obj *storage.NetworkPolicy) error
	Delete(ctx context.Context, id ...string) error

	Walk(ctx context.Context, fn func(obj *storage.NetworkPolicy) error) error
}
