package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// EntityStore stores network graph entities.
//
//go:generate mockgen-wrapper
type EntityStore interface {
	Exists(ctx context.Context, id string) (bool, error)

	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.NetworkEntity, bool, error)

	Upsert(ctx context.Context, entity *storage.NetworkEntity) error
	UpsertMany(ctx context.Context, objs []*storage.NetworkEntity) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	Walk(ctx context.Context, fn func(obj *storage.NetworkEntity) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.NetworkEntity) error) error

	// Deprecated: use GetByQueryFn instead
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.NetworkEntity, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.NetworkEntity) error) error
}
