package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process indicators.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, []int, error)

	UpsertMany(context.Context, []*storage.ProcessIndicator) error
	DeleteMany(ctx context.Context, id []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)

	Walk(context.Context, func(pi *storage.ProcessIndicator) error) error
	DeleteByQuery(ctx context.Context, query *v1.Query) error
}
