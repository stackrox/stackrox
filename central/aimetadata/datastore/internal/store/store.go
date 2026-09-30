package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides persistence for AI metadata associated with custom resources.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.AIMetadata, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.AIMetadata, error)
	Upsert(ctx context.Context, metadata *storage.AIMetadata) error
	Delete(ctx context.Context, id string) error
}
