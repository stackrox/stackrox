package cachedstore

import (
	"bitbucket.org/stack-rox/apollo/central/authprovider/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
)

// CachedStore is an intermediary to storage that keeps an in-memory cache of auth providers.
type CachedStore interface {
	// GetAuthenticators gets the cached map from id to the authenticator object.
	GetAuthenticators() map[string]authproviders.Authenticator
	// RefreshCache discards the cache and regenerates it from the persistent store.
	RefreshCache()

	// The following methods just proxy through to the underlying store.
	GetAuthProvider(id string) (*v1.AuthProvider, bool, error)
	GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error)
	AddAuthProvider(authProvider *v1.AuthProvider) (string, error)
	UpdateAuthProvider(authProvider *v1.AuthProvider) error
	RemoveAuthProvider(id string) error
	RecordAuthSuccess(id string) error
}

// New returns a new cached store.
func New(store store.Store) (cachedStore CachedStore) {
	cachedStore = &cachedStoreImpl{
		store: store,
	}
	// This will make sure the cache is up-to-date on restarts.
	cachedStore.RefreshCache()
	return
}
