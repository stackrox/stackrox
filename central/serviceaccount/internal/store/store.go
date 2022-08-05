package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the service account store interface
type Store interface {
	Get(ctx context.Context, id string) (*storage.ServiceAccount, bool, error)
	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ServiceAccount, []int, error)
	Walk(context.Context, func(sa *storage.ServiceAccount) error) error

	Upsert(ctx context.Context, serviceaccount *storage.ServiceAccount) error
	Delete(ctx context.Context, id string) error
}
