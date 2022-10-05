package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	Walk(context.Context, func(namespace *storage.NamespaceMetadata) error) error
	Upsert(context.Context, *storage.NamespaceMetadata) error
	Delete(ctx context.Context, id string) error
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.NamespaceMetadata, error)
}
