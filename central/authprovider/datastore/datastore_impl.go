package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	authProviderSAC = sac.ForResource(resources.AuthProvider)
)

type datastoreImpl struct {
	storage store.Store
}

// GetAuthProviders retrieves authProviders from bolt
func (b *datastoreImpl) GetAllAuthProviders() ([]*storage.AuthProvider, error) {
	// No SAC checks here because all users need to be able to read auth providers in order to authenticate.
	return b.storage.GetAllAuthProviders()
}

// AddAuthProvider adds an auth provider into bolt
func (b *datastoreImpl) AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.AddAuthProvider(authProvider)
}

// UpdateAuthProvider upserts an auth provider into bolt
func (b *datastoreImpl) UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.UpdateAuthProvider(authProvider)
}

// RemoveAuthProvider removes an auth provider from bolt
func (b *datastoreImpl) RemoveAuthProvider(ctx context.Context, id string) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.RemoveAuthProvider(id)
}
