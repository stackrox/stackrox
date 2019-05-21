package authproviders

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for auth providers.
type Store interface {
	GetAllAuthProviders() ([]*storage.AuthProvider, error)

	AddAuthProvider(authProvider *storage.AuthProvider) error
	UpdateAuthProvider(authProvider *storage.AuthProvider) error
	RemoveAuthProvider(id string) error
}
