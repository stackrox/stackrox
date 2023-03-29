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
	"github.com/stackrox/rox/pkg/logging"
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

var (
	log = logging.LoggerForModule()
)

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

func (u *authProviderUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	authProviderIDsToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	authProviders, err := u.authProviderDS.GetAuthProvidersFiltered(ctx, func(authProvider *storage.AuthProvider) bool {
		return utils.IsDeclarativeOrigin(authProvider.GetTraits().GetOrigin()) &&
			!authProviderIDsToSkip.Contains(authProvider.GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative auth providers")
	}

	var authProviderDeletionErr *multierror.Error
	var authProviderIDs []string
	for _, authProvider := range authProviders {
		referencingGroups, err := u.groupDS.GetFiltered(ctx, func(group *storage.Group) bool {
			return group.GetProps().GetAuthProviderId() == authProvider.GetId()
		})
		if err != nil || len(referencingGroups) > 0 {
			if err == nil {
				err = errox.ReferencedByAnotherObject.Newf("auth provider is still referenced by groups")
			}
			authProviderDeletionErr = multierror.Append(authProviderDeletionErr, err)
			authProviderIDs = append(authProviderIDs, authProvider.GetId())

			u.reporter.UpdateIntegrationHealthAsync(utils.IntegrationHealthForProtoMessage(authProvider, "", err,
				u.idExtractor, u.nameExtractor))
			authProvider.Traits.Origin = storage.Traits_DECLARATIVE_ORPHANED
			if err = u.authProviderDS.UpdateAuthProvider(ctx, authProvider); err != nil {
				authProviderDeletionErr = multierror.Append(authProviderDeletionErr, err)
			}
			continue
		}
		if err := u.authProviderRegistry.DeleteProvider(ctx, authProvider.GetId(), true, true); err != nil {
			authProviderDeletionErr = multierror.Append(authProviderDeletionErr, err)
			authProviderIDs = append(authProviderIDs, authProvider.GetId())

			u.reporter.UpdateIntegrationHealthAsync(utils.IntegrationHealthForProtoMessage(authProvider, "", err,
				u.idExtractor, u.nameExtractor))
		}
	}
	return authProviderIDs, authProviderDeletionErr.ErrorOrNil()
}
