package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides persistence for tracked Kubernetes custom resources.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.CustomResource, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.CustomResource, error)
	Upsert(ctx context.Context, customResource *storage.CustomResource) error
	Delete(ctx context.Context, id string) error
}
