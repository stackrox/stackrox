package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	Walk(ctx context.Context, fn func(*storage.Alert) error) error
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.Alert, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error)
	Upsert(ctx context.Context, alert *storage.Alert) error
	UpsertMany(ctx context.Context, alerts []*storage.Alert) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}
