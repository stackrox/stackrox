package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	authProviderSAC = sac.ForResource(resources.AuthProvider)
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

// GetAllAuthProviders retrieves authProviders.
func (b *datastoreImpl) GetAllAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error) {
	// No SAC checks here because all users need to be able to read auth providers in order to authenticate.
	return b.storage.GetAll(ctx)
}

// AddAuthProvider adds an auth provider into bolt.
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

// UpdateAuthProvider upserts an auth provider into bolt.
func (b *datastoreImpl) UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	// The API currently does not allow setting the force flag since it would require a non-compatible gRPC change.
	if err := b.verifyExistsAndMutable(ctx, authProvider.GetId(), false); err != nil {
		return err
	}
	return b.storage.Upsert(ctx, authProvider)
}

// RemoveAuthProvider removes an auth provider from bolt.
func (b *datastoreImpl) RemoveAuthProvider(ctx context.Context, id string, force bool) error {
	if ok, err := authProviderSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := b.verifyExistsAndMutable(ctx, id, force); err != nil {
		return err
	}
	return b.storage.Delete(ctx, id)
}

func (b *datastoreImpl) verifyExistsAndMutable(ctx context.Context, id string, force bool) error {
	provider, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return errox.NotFound.Newf("auth provider with id %q was not found", id)
	}

	switch provider.GetTraits().GetMutabilityMode() {
	case storage.Traits_ALLOW_MUTATE:
		return nil
	case storage.Traits_ALLOW_MUTATE_FORCED:
		if force {
			return nil
		}
		return errox.InvalidArgs.Newf("auth provider %q is immutable and can only be removed"+
			" via API and specifying the force flag", id)
	default:
		utils.Should(errors.Wrapf(errox.InvalidArgs, "unknown mutability mode given: %q",
			provider.GetTraits().GetMutabilityMode()))
	}
	return errox.InvalidArgs.Newf("auth provider %q is immutable", id)
}
