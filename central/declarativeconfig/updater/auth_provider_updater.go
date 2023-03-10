package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/utils"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/set"
)

type authProviderUpdater struct {
	authProviderDS       authproviders.Store
	authProviderRegistry authproviders.Registry
	groupDS              groupDataStore.DataStore
	reporter             integrationhealth.Reporter
	idExtractor          types.IDExtractor
	nameExtractor        types.NameExtractor
}

var _ ResourceUpdater = (*authProviderUpdater)(nil)

func newAuthProviderUpdater(authProvidersDS authproviders.Store, registry authproviders.Registry,
	groupDS groupDataStore.DataStore, reporter integrationhealth.Reporter) ResourceUpdater {
	return &authProviderUpdater{
		authProviderDS:       authProvidersDS,
		authProviderRegistry: registry,
		groupDS:              groupDS,
		reporter:             reporter,
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

func (u *authProviderUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) error {
	resourcesToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	authProviders, err := u.authProviderDS.GetAuthProvidersFiltered(ctx, func(authProvider *storage.AuthProvider) bool {
		return authProvider.GetTraits().GetOrigin() == storage.Traits_DECLARATIVE &&
			!resourcesToSkip.Contains(authProvider.GetId())
	})
	if err != nil {
		return errors.Wrap(err, "retrieving declarative auth providers")
	}

	var authProviderDeletionErr *multierror.Error
	for _, authProvider := range authProviders {
		if err := u.authProviderRegistry.DeleteProvider(ctx, authProvider.GetId(), true, true); err != nil {
			authProviderDeletionErr = multierror.Append(authProviderDeletionErr, err)

			u.reporter.UpdateIntegrationHealthAsync(utils.IntegrationHealthForProtoMessage(authProvider, "", err,
				u.idExtractor, u.nameExtractor))
			continue
		}
		// TODO(ROX-14700): This currently deletes also declarative groups and should resolve these references.
		if err := u.groupDS.RemoveAllWithAuthProviderID(ctx, authProvider.GetId(), true); err != nil {
			log.Errorf("Error deleting groups for auth provider id %s: %v", authProvider.GetId(), err)
		}
		if err := u.reporter.RemoveIntegrationHealth(u.idExtractor(authProvider)); err != nil {
			log.Errorf("Error removing the health status for auth provider %s: %v", authProvider.GetId(), err)
		}
	}
	return authProviderDeletionErr.ErrorOrNil()
}
