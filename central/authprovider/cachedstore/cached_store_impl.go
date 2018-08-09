package cachedstore

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/authprovider/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

type cachedStoreImpl struct {
	store store.Store
	cache map[string]authproviders.AuthProvider
	lock  sync.Mutex
}

// GetParsedAuthProviders gets the cached map from id to the authenticator object.
func (c *cachedStoreImpl) GetParsedAuthProviders() map[string]authproviders.AuthProvider {
	c.lock.Lock()
	defer c.lock.Unlock()
	clone := make(map[string]authproviders.AuthProvider)
	for k, v := range c.cache {
		clone[k] = v
	}
	return clone
}

// RefreshCache discards and regenerates the cache.
func (c *cachedStoreImpl) RefreshCache() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache = make(map[string]authproviders.AuthProvider)

	providers, err := c.GetAuthProviders(&v1.GetAuthProvidersRequest{})
	if err != nil {
		logger.Errorf("RefreshCache: error retrieving auth providers: %s", err)
		return
	}

	for _, provider := range providers {
		authenticator, err := authproviders.Create(provider)
		if err != nil {
			logger.Errorf("RefreshCache: error creating auth provider for %#v: %s", provider, err)
			continue
		}
		c.cache[provider.GetId()] = authenticator
	}
}

// addToCache adds an authProvider to the cache.
// The lock MUST be held when calling this function.
func (c *cachedStoreImpl) addToCache(id string, authProvider *v1.AuthProvider) error {
	authenticator, err := authproviders.Create(authProvider)
	if err != nil {
		return fmt.Errorf("authenticator creation: %s", err)
	}
	c.cache[id] = authenticator
	return nil
}

// AddAuthProvider is a pass-through to the underlying store, which
// also updates our cache.
func (c *cachedStoreImpl) AddAuthProvider(authProvider *v1.AuthProvider) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	id, err := c.store.AddAuthProvider(authProvider)
	if err != nil {
		return "", fmt.Errorf("insertion to storage: %s", err)
	}
	err = c.addToCache(id, authProvider)
	if err != nil {
		return "", fmt.Errorf("insertion to cache: %s", err)
	}
	return id, nil
}

// GetAuthProvider is a pass-through to the underlying store.
func (c *cachedStoreImpl) GetAuthProvider(id string) (*v1.AuthProvider, bool, error) {
	return c.store.GetAuthProvider(id)
}

// GetAuthProviders is a pass-through to the underlying store.
func (c *cachedStoreImpl) GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error) {
	return c.store.GetAuthProviders(request)
}

// RecordAuthSuccess is a pass-through to the underlying store, which also takes care of updating the cache.
func (c *cachedStoreImpl) RecordAuthSuccess(id string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.store.RecordAuthSuccess(id)
	if err != nil {
		return err
	}
	// Now update the cached entry.
	authProvider, exists, err := c.GetAuthProvider(id)
	if err != nil {
		return fmt.Errorf("retrieving just-updated auth provider: %s", err)
	}
	if !exists {
		return fmt.Errorf("auth provider %s doesn't exist, but we didn't throw an error when recording auth success", id)
	}
	err = c.addToCache(id, authProvider)
	if err != nil {
		return fmt.Errorf("insertion to cache: %s", err)
	}
	return nil
}

// RemoveAuthProvider is a pass-through to the underlying store.
func (c *cachedStoreImpl) RemoveAuthProvider(id string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.store.RemoveAuthProvider(id)
	if err != nil {
		return fmt.Errorf("removing from store: %s", err)
	}

	delete(c.cache, id)
	return nil
}

// UpdateAuthProvider is a pass-through to the underlying store.
func (c *cachedStoreImpl) UpdateAuthProvider(authProvider *v1.AuthProvider) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.store.UpdateAuthProvider(authProvider)
	if err != nil {
		return fmt.Errorf("insertion to storage: %s", err)
	}

	err = c.addToCache(authProvider.GetId(), authProvider)
	if err != nil {
		return fmt.Errorf("insertion to cache: %s", err)
	}
	return nil
}
