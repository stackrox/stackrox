package authproviders

import "github.com/stackrox/rox/generated/api/v1"

// Store provides storage functionality for auth providers.
type Store interface {
	GetAllAuthProviders() ([]*v1.AuthProvider, error)

	AddAuthProvider(authProvider *v1.AuthProvider) error
	UpdateAuthProvider(authProvider *v1.AuthProvider) error
	RemoveAuthProvider(id string) error
	RecordAuthSuccess(id string) error
}
