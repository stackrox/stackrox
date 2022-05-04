package datastore

import (
	"context"

	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	authProviderSAC = sac.ForResource(resources.AuthProvider)
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

// GetAllAuthProviders retrieves authProviders
func (b *datastoreImpl) GetAllAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error) {
	// No SAC checks here because all users need to be able to read auth providers in order to authenticate.
	return b.storage.GetAll(ctx)
}

// AddAuthProvider adds an auth provider into bolt
func (b *datastoreImpl) AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	exists, err := b.storage.Exists(ctx, authProvider.GetId())
	if err != nil {
		return err
	}
	if exists {
		return errox.InvalidArgs.Newf("auth provider with id %q was found", authProvider.GetId())
	}
	return b.storage.Upsert(ctx, authProvider)
}

// UpdateAuthProvider upserts an auth provider into bolt
func (b *datastoreImpl) UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	exists, err := b.storage.Exists(ctx, authProvider.GetId())
	if err != nil {
		return err
	}
	if !exists {
		return errox.NotFound.Newf("auth provider with id %q was not found", authProvider.GetId())
	}
	return b.storage.Upsert(ctx, authProvider)
}

// RemoveAuthProvider removes an auth provider from bolt
func (b *datastoreImpl) RemoveAuthProvider(ctx context.Context, id string) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.storage.Delete(ctx, id)
}
