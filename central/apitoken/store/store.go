package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	apiTokensBucket = "apiTokens"
)

// Store is the (bolt-backed) store for API tokens.
// We don't store the tokens themselves, but do store metadata.
// Importantly, the Store persists token revocations.
type Store interface {
	AddToken(*v1.TokenMetadata) error
	GetTokenOrNil(id string) (token *v1.TokenMetadata, err error)
	GetTokens(*v1.GetAPITokensRequest) ([]*v1.TokenMetadata, error)
	RevokeToken(id string) (exists bool, err error)
}

// New returns a ready-to-use store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, apiTokensBucket)
	return &storeImpl{DB: db}
}
