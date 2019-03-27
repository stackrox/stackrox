package store

import (
	bolt "github.com/etcd-io/bbolt"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	apiTokensBucket = []byte("apiTokens")
)

// Store is the (bolt-backed) store for API tokens.
// We don't store the tokens themselves, but do store metadata.
// Importantly, the Store persists token revocations.
type Store interface {
	AddToken(*storage.TokenMetadata) error
	GetTokenOrNil(id string) (token *storage.TokenMetadata, err error)
	GetTokens(*v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)
	RevokeToken(id string) (exists bool, err error)
}

// New returns a ready-to-use store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, apiTokensBucket)
	return &storeImpl{DB: db}
}
