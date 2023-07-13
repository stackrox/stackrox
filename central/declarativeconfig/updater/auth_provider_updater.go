package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

type authProviderUpdater struct {
	authProviderDS       authproviders.Store
	authProviderRegistry authproviders.Registry
	groupDS              groupDataStore.DataStore
	healthDS             declarativeConfigHealth.DataStore
	idExtractor          types.IDExtractor
	nameExtractor        types.NameExtractor
}

var _ ResourceUpdater = (*authProviderUpdater)(nil)

var (
	log                       = logging.LoggerForModule()
	deleteImperativeGroupsCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
)

func newAuthProviderUpdater(authProvidersDS authproviders.Store, registry authproviders.Registry,
	groupDS groupDataStore.DataStore, healthDS declarativeConfigHealth.DataStore) ResourceUpdater {
	return &authProviderUpdater{
		authProviderDS:       authProvidersDS,
		authProviderRegistry: registry,
		groupDS:              groupDS,
		healthDS:             healthDS,
		idExtractor:          types.UniversalIDExtractor(),
		nameExtractor:        types.UniversalNameExtractor(),
	}
}

func (u *authProviderUpdater) Upsert(ctx context.Context, m proto.Message) error {
	authProvider, ok := m.(*storage.AuthProvider)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to auth provider updater: %T", authProvider)
	}
	if err := u.authProviderRegistry.DeleteProvider(ctx, authProvider.GetId(), true, true); err != nil {
		return err
	}
	if _, err := u.authProviderRegistry.CreateProvider(ctx, authproviders.WithStorageView(authProvider),
		authproviders.WithAttributeVerifier(authProvider),
		authproviders.WithValidateCallback(authProviderDatastore.Singleton())); err != nil {
		return err
	}
	return nil
}

func (u *authProviderUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	authProviderIDsToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	authProviders, err := u.authProviderDS.GetAuthProvidersFiltered(ctx, func(authProvider *storage.AuthProvider) bool {
		return declarativeconfig.IsDeclarativeOrigin(authProvider) &&
			!authProviderIDsToSkip.Contains(authProvider.GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative auth providers")
	}

	var authProviderDeletionErr *multierror.Error
	authProviderIDs := set.NewStringSet()
	for _, authProvider := range authProviders {
		referencingGroups, err := u.groupDS.GetFiltered(ctx, func(group *storage.Group) bool {
			return group.GetProps().GetAuthProviderId() == authProvider.GetId()
		})
		if err != nil {
			authProviderDeletionErr, authProviderIDs = u.processDeletionError(ctx, authProviderDeletionErr, err, authProviderIDs, authProvider)
			continue
		}
		var hasErrorDeletingGroups bool
		for _, group := range referencingGroups {
			ctxToUse := utils.IfThenElse(declarativeconfig.IsDeclarativeOrigin(group.GetProps()), ctx, deleteImperativeGroupsCtx)
			if err = u.groupDS.Remove(ctxToUse, group.GetProps(), true); err != nil {
				authProviderDeletionErr, authProviderIDs = u.processDeletionError(ctx, authProviderDeletionErr, err, authProviderIDs, authProvider)
				hasErrorDeletingGroups = true
			}
		}
		// Since group is valid only if it references existing auth provider, we can't delete auth provider
		// until all referencing groups are successfully deleted.
		if hasErrorDeletingGroups {
			continue
		}
		if err := u.authProviderRegistry.DeleteProvider(ctx, authProvider.GetId(), true, true); err != nil {
			authProviderDeletionErr, authProviderIDs = u.processDeletionError(ctx, authProviderDeletionErr, err, authProviderIDs, authProvider)
		}
	}
	return authProviderIDs.AsSlice(), authProviderDeletionErr.ErrorOrNil()
}

func (u *authProviderUpdater) processDeletionError(ctx context.Context, authProviderDeletionErr *multierror.Error,
	err error, authProviderIDs set.Set[string], authProvider *storage.AuthProvider) (*multierror.Error, set.Set[string]) {
	authProviderDeletionErr = multierror.Append(authProviderDeletionErr, err)
	authProviderIDs.Add(authProvider.GetId())

	if err := u.healthDS.UpdateStatusForDeclarativeConfig(ctx, u.idExtractor(authProvider), err); err != nil {
		log.Errorf("Failed to update the declarative config health status %q: %v", authProvider.GetId(), err)
	}
	return authProviderDeletionErr, authProviderIDs
}
