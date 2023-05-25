package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides the interface to the underlying storage
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.NetworkPolicy, error)
	Upsert(ctx context.Context, obj *storage.NetworkPolicy) error
	Delete(ctx context.Context, id string) error

	Walk(ctx context.Context, fn func(obj *storage.NetworkPolicy) error) error
}
