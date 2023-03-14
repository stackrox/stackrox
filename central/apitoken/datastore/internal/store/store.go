package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the (bolt-backed) store for API tokens.
// We don't store the tokens themselves, but do store metadata.
// Importantly, the Store persists token revocations.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.TokenMetadata, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.TokenMetadata, error)
	Walk(context.Context, func(*storage.TokenMetadata) error) error
	Upsert(context.Context, *storage.TokenMetadata) error
}
