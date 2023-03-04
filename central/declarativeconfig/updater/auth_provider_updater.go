package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
)

type authProviderUpdater struct {
	authProviderDS       authproviders.Store
	authProviderRegistry authproviders.Registry
}

var _ ResourceUpdater = (*authProviderUpdater)(nil)

func newAuthProviderUpdater(datastore authproviders.Store, registry authproviders.Registry) ResourceUpdater {
	return &authProviderUpdater{
		authProviderDS:       datastore,
		authProviderRegistry: registry,
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
