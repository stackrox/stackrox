package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store is the store for API tokens.
// We don't store the tokens themselves, but do store metadata.
// Importantly, the Store persists token revocations.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.TokenMetadata, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.TokenMetadata, error)
	Walk(context.Context, func(*storage.TokenMetadata) error) error
	Upsert(context.Context, *storage.TokenMetadata) error
	Delete(ctx context.Context, id string) error
}
