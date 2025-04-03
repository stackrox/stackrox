package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for active component.
//
//go:generate mockgen-wrapper
type Store interface {
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ActiveComponent, []int, error)
	UpsertMany(ctx context.Context, activeComponents []*storage.ActiveComponent) error
	Delete(ctx context.Context, id ...string) error
}
