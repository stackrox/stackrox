package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the cloud sources data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.CloudSource, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.CloudSource, error)
	Upsert(ctx context.Context, obj *storage.CloudSource) error
	Delete(ctx context.Context, id string) error
}
