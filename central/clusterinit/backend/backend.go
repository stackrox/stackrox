package backend

import (
	"context"

	"github.com/stackrox/rox/central/clusterinit/datastore"
	"github.com/stackrox/rox/generated/storage"
)

// Backend is the backend for the bootstrap-tokens component.
type Backend interface {
	GetAll(ctx context.Context) ([]*storage.BootstrapTokenWithMeta, error)
	Get(ctx context.Context, tokenID string) (*storage.BootstrapTokenWithMeta, error)
	Issue(ctx context.Context, description string) (*storage.BootstrapTokenWithMeta, error)
	Revoke(ctx context.Context, tokenID string) error
	SetActive(ctx context.Context, tokenID string, active bool) error
}

func newBackend(tokenStore datastore.DataStore) Backend {
	return &backendImpl{
		tokenStore: tokenStore,
	}
}
