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
	Get(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	Walk(context.Context, func(namespace *storage.NamespaceMetadata) error) error
	Upsert(context.Context, *storage.NamespaceMetadata) error
	Delete(ctx context.Context, id ...string) error
	GetMany(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, []int, error)
}
