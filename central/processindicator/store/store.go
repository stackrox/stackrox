package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for process indicators.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error)
	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, []int, error)

	UpsertMany(context.Context, []*storage.ProcessIndicator) error
	Delete(ctx context.Context, id ...string) error

	Walk(context.Context, func(pi *storage.ProcessIndicator) error) error
	DeleteByQuery(ctx context.Context, query *v1.Query) ([]string, error)
}
