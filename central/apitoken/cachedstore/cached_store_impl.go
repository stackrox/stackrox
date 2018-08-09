package cachedstore

import (
	"fmt"

	"github.com/deckarep/golang-set"
	"github.com/stackrox/rox/central/apitoken/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type cachedStoreImpl struct {
	store store.Store

	revokedTokens mapset.Set
}

func (c *cachedStoreImpl) AddToken(token *v1.TokenMetadata) error {
	return c.store.AddToken(token)
}

func (c *cachedStoreImpl) GetToken(id string) (token *v1.TokenMetadata, exists bool, err error) {
	return c.store.GetToken(id)
}

func (c *cachedStoreImpl) GetTokens(req *v1.GetAPITokensRequest) ([]*v1.TokenMetadata, error) {
	return c.store.GetTokens(req)
}

func (c *cachedStoreImpl) RevokeToken(id string) (exists bool, err error) {
	exists, err = c.store.RevokeToken(id)
	if !exists || err != nil {
		return
	}
	c.revokedTokens.Add(id)
	return
}

func (c *cachedStoreImpl) CheckTokenRevocation(id string) error {
	if c.revokedTokens.Contains(id) {
		return fmt.Errorf("token '%s' is revoked", id)
	}
	return nil
}
