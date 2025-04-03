package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store encapsulates the service account store interface
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.ServiceAccount, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ServiceAccount, []int, error)
	Walk(context.Context, func(sa *storage.ServiceAccount) error) error

	Upsert(ctx context.Context, serviceaccount *storage.ServiceAccount) error
	Delete(ctx context.Context, id ...string) error
}
