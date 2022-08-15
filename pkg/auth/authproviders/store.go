package authproviders

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for auth providers.
type Store interface {
	GetAllAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error)

	AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error
	RemoveAuthProvider(ctx context.Context, deleteReq *v1.DeleteByIDWithForce) error
}
