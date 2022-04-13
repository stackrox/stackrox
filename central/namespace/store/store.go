package store

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	Walk(context.Context, func(namespace *storage.NamespaceMetadata) error) error
	Upsert(context.Context, *storage.NamespaceMetadata) error
	Delete(ctx context.Context, id string) error
}
