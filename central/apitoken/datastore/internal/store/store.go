package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store is the (bolt-backed) store for API tokens.
// We don't store the tokens themselves, but do store metadata.
// Importantly, the Store persists token revocations.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.TokenMetadata, bool, error)
	Walk(func(*storage.TokenMetadata) error) error
	Upsert(*storage.TokenMetadata) error
}
