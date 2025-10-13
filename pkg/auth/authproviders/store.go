package authproviders

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for auth providers.
//
//go:generate mockgen-wrapper
type Store interface {
	GetAuthProvider(ctx context.Context, id string) (*storage.AuthProvider, bool, error)
	ForEachAuthProvider(ctx context.Context, fn func(obj *storage.AuthProvider) error) error
	GetAuthProvidersFiltered(ctx context.Context, filter func(authProvider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error)
	AuthProviderExistsWithName(ctx context.Context, name string) (bool, error)
	AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	RemoveAuthProvider(ctx context.Context, id string, force bool) error
}
