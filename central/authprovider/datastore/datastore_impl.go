package datastore

import (
	"context"
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
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

// GetAllAuthProviders retrieves authProviders and process each one with provided function.
func (b *datastoreImpl) ProcessAuthProviders(ctx context.Context, fn func(obj *storage.AuthProvider) error) error {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return err
	}

	return b.storage.Walk(ctx, fn)
}

func (b *datastoreImpl) GetAuthProvider(ctx context.Context, id string) (*storage.AuthProvider, bool, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}

	return b.storage.Get(ctx, id)
}
func (b *datastoreImpl) AuthProviderExistsWithName(ctx context.Context, name string) (bool, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return false, err
	}

	query := search.NewQueryBuilder().AddExactMatches(search.AuthProviderName, name).ProtoQuery()
	results, err := b.storage.Search(ctx, query)
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

func (b *datastoreImpl) GetAuthProvidersFiltered(ctx context.Context,
	filter func(provider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	var filteredAuthProviders []*storage.AuthProvider
	err := b.storage.Walk(ctx, func(authProvider *storage.AuthProvider) error {
		if filter(authProvider) {
			filteredAuthProviders = append(filteredAuthProviders, authProvider)
		}
		return nil
	})
	if err != nil {
		return nil, pkgErrors.Wrap(err, "retrieving auth providers")
	}
	return filteredAuthProviders, nil
}

// AddAuthProvider adds an auth provider into the database.
func (b *datastoreImpl) AddAuthProvider(ctx context.Context, authProvider *storage.AuthProvider) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := verifyAuthProviderOrigin(ctx, authProvider); err != nil {
		return pkgErrors.Wrap(err, "origin didn't match for new auth provider")
	}
	if err := validateAuthProvider(authProvider); err != nil {
		return err
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
	if err := validateAuthProvider(authProvider); err != nil {
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
		return pkgErrors.Wrap(err, "origin didn't match for existing auth provider")
	}
	if err = verifyAuthProviderOrigin(ctx, authProvider); err != nil {
		return pkgErrors.Wrap(err, "origin didn't match for new auth provider")
	}
	return b.storage.Upsert(ctx, authProvider)
}

// RemoveAuthProvider removes an auth provider from the database.
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
		return pkgErrors.Wrapf(errox.NotAuthorized, "auth provider %q's origin is %s, cannot be modified or deleted with the current permission",
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
		utils.Should(pkgErrors.Wrapf(errox.InvalidArgs, "unknown mutability mode given: %q",
			provider.GetTraits().GetMutabilityMode()))
	}
	return nil, errox.InvalidArgs.Newf("auth provider %q is immutable", id)
}

func validateAuthProvider(ap *storage.AuthProvider) error {
	var validationErrs error
	if ap.GetId() == "" {
		validationErrs = errors.Join(validationErrs, errox.InvalidArgs.CausedBy("auth provider ID is empty"))
	}
	if ap.GetName() == "" {
		validationErrs = errors.Join(validationErrs, errox.InvalidArgs.CausedBy("auth provider name is empty"))
	}
	if ap.GetLoginUrl() == "" {
		validationErrs = errors.Join(validationErrs, errox.InvalidArgs.CausedBy("auth provider login URL is empty"))
	}
	return pkgErrors.Wrap(validationErrs, "validating auth provider")
}
