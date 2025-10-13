package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store stores and retrieves providers from the KV storage mechanism.
//
//go:generate mockgen-wrapper
type Store interface {
	Walk(ctx context.Context, fn func(obj *storage.AuthProvider) error) error
	Get(ctx context.Context, id string) (*storage.AuthProvider, bool, error)

	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Exists(ctx context.Context, id string) (bool, error)
	Upsert(ctx context.Context, obj *storage.AuthProvider) error
	Delete(ctx context.Context, id string) error
}
