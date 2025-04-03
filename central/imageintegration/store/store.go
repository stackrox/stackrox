package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for alerts.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.ImageIntegration, bool, error)
	GetAll(ctx context.Context) ([]*storage.ImageIntegration, error)
	Upsert(ctx context.Context, integration *storage.ImageIntegration) error
	UpsertMany(ctx context.Context, objs []*storage.ImageIntegration) error
	Delete(ctx context.Context, id ...string) error
	PruneMany(ctx context.Context, identifiers []string) error
}
