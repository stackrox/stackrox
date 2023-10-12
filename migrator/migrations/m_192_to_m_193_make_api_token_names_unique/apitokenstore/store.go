package apitokenstore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to interact with the storage for storage.TokenMetadata
//
//go:generate mockgen-wrapper
type Store interface {
	UpsertMany(ctx context.Context, objs []*storage.TokenMetadata) error

	Walk(ctx context.Context, fn func(obj *storage.TokenMetadata) error) error
}
