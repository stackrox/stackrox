package authproviders

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for auth providers.
type Store interface {
	GetAllAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error)
	GetAuthProvidersFiltered(ctx context.Context, filter func(authProvider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error)
	AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	RemoveAuthProvider(ctx context.Context, id string, force bool) error
}
