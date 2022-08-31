package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for active component.
//
//go:generate mockgen-wrapper
type Store interface {
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ActiveComponent, []int, error)
	UpsertMany(ctx context.Context, activeComponents []*storage.ActiveComponent) error
	DeleteMany(ctx context.Context, ids []string) error
}
