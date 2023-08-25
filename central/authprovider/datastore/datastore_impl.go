package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	accessSAC = sac.ForResource(resources.Access)
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

// GetAllAuthProviders retrieves authProviders.
func (b *datastoreImpl) GetAllAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	return b.storage.GetAll(ctx)
}

func (b *datastoreImpl) GetAuthProvider(ctx context.Context, id string) (*storage.AuthProvider, bool, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}

	return b.storage.Get(ctx, id)
}

func (b *datastoreImpl) GetAuthProvidersFiltered(ctx context.Context,
	filter func(provider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	// TODO(ROX-15902): The store currently doesn't provide a Walk function. This is mostly due to us supporting the
	// old bolt store. Once we deprecate old store solutions with the 4.0.0 release, this should be changed to use
	// store.Walk.
	authProviders, err := b.storage.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving auth providers")
	}
	filteredAuthProviders := make([]*storage.AuthProvider, 0, len(authProviders))
	for _, authProvider := range authProviders {
		if filter(authProvider) {
			filteredAuthProviders = append(filteredAuthProviders, authProvider)
		}
	}
	return filteredAuthProviders, nil
}

// AddAuthProvider adds an auth provider into bolt.
func (b *datastoreImpl) AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := verifyAuthProviderOrigin(ctx, authProvider); err != nil {
		return errors.Wrap(err, "origin didn't match for new auth provider")
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

// UpdateAuthProvider upserts an auth provider.
func (b *datastoreImpl) UpdateAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	// Currently, the data store does not support forcing updates.
	// If we want to add a force flag to the respective API methods, we might need to revisit this.
	existingProvider, err := b.verifyExistsAndMutable(ctx, authProvider.GetId(), false)
	if err != nil {
		return err
	}
	if err = verifyAuthProviderOrigin(ctx, existingProvider); err != nil {
		return errors.Wrap(err, "origin didn't match for existing auth provider")
	}
	if err = verifyAuthProviderOrigin(ctx, authProvider); err != nil {
		return errors.Wrap(err, "origin didn't match for new auth provider")
	}
	return b.storage.Upsert(ctx, authProvider)
}

// RemoveAuthProvider removes an auth provider from bolt.
func (b *datastoreImpl) RemoveAuthProvider(ctx context.Context, id string, force bool) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	ap, err := b.verifyExistsAndMutable(ctx, id, force)
	if err != nil {
		return err
	}
	if err = verifyAuthProviderOrigin(ctx, ap); err != nil {
		return err
	}
	return b.storage.Delete(ctx, id)
}

func verifyAuthProviderOrigin(ctx context.Context, ap *storage.AuthProvider) error {
	if !declarativeconfig.CanModifyResource(ctx, ap) {
		return errors.Wrapf(errox.NotAuthorized, "auth provider %q's origin is %s, cannot be modified or deleted with the current permission",
			ap.GetName(), ap.GetTraits().GetOrigin())
	}
	return nil
}

func (b *datastoreImpl) verifyExistsAndMutable(ctx context.Context, id string, force bool) (*storage.AuthProvider, error) {
	provider, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("auth provider with id %q was not found", id)
	}

	switch provider.GetTraits().GetMutabilityMode() {
	case storage.Traits_ALLOW_MUTATE:
		return provider, nil
	case storage.Traits_ALLOW_MUTATE_FORCED:
		if force {
			return provider, nil
		}
		return nil, errox.InvalidArgs.Newf("auth provider %q is immutable and can only be removed"+
			" via API and specifying the force flag", id)
	default:
		utils.Should(errors.Wrapf(errox.InvalidArgs, "unknown mutability mode given: %q",
			provider.GetTraits().GetMutabilityMode()))
	}
	return nil, errox.InvalidArgs.Newf("auth provider %q is immutable", id)
}
