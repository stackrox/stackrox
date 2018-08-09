package cachedstore

import (
	"github.com/deckarep/golang-set"
	"github.com/stackrox/rox/central/apitoken/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// CachedStore is the access point to the API token's store.
// Tokens will be revoked relatively infrequently, but the tokens will be
// queried for revocation frequently, and the cached store maintains a layer of
// data structures optimized for this.
type CachedStore interface {
	AddToken(*v1.TokenMetadata) error
	GetToken(id string) (token *v1.TokenMetadata, exists bool, err error)
	GetTokens(*v1.GetAPITokensRequest) ([]*v1.TokenMetadata, error)
	RevokeToken(id string) (exists bool, err error)

	CheckTokenRevocation(id string) error
}

// New returns a ready-to-use CachedStore.
func New(store store.Store) (CachedStore, error) {
	c := &cachedStoreImpl{store: store}
	err := c.initializeFromStore()
	if err != nil {
		return nil, err
	}
	return c, nil
}

// initializeFromStore syncs the in-mem cache with the underlying storage.
func (c *cachedStoreImpl) initializeFromStore() error {
	c.revokedTokens = mapset.NewSet()

	tokens, err := c.store.GetTokens(&v1.GetAPITokensRequest{})
	if err != nil {
		return err
	}

	for _, token := range tokens {
		if token.GetRevoked() {
			c.revokedTokens.Add(token.GetId())
		}
	}

	return nil
}
