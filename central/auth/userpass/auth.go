package userpass

import (
	"context"
	"time"

	"github.com/pkg/errors"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	basicAuthProvider "github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	basicAuthn "github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	htpasswdDir  = "/run/secrets/stackrox.io/htpasswd"
	htpasswdFile = "htpasswd"

	watchInterval = 5 * time.Second
)

var (
	log = logging.LoggerForModule()

	// basicAuthProviderID is the auth provider ID used for basic auth. This is arbitrary, but should not be changed.
	basicAuthProviderID = "4df1b98c-24ed-4073-a9ad-356aec6bb62d"
)

// CreateManager creates and returns a manager for user/password authentication.
func CreateManager(store roleDatastore.DataStore) (*basicAuthn.Manager, error) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	adminRole, found, err := store.GetRole(ctx, accesscontrol.Admin)
	if err != nil || !found || adminRole == nil {
		return nil, errors.Wrap(err, "Could not look up admin role")
	}

	mgr := basicAuthn.NewManager(nil, mapper.AlwaysAdminRoleMapper())

	wh := &watchHandler{
		manager: mgr,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}

	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), htpasswdDir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)

	return mgr, nil
}

// RegisterAuthProviderOrPanic sets up basic authentication with the builtin htpasswd file. It panics if the basic auth
// feature is not enabled, or if it is called twice on the same registry.
func RegisterAuthProviderOrPanic(ctx context.Context, mgr *basicAuthn.Manager, registry authproviders.Registry) authproviders.Provider {
	err := registry.RegisterBackendFactory(ctx, basicAuthProvider.TypeName, basicAuthProvider.NewFactory)
	if err != nil {
		log.Warnf("Could not register basic auth provider factory: %v", err)
	}

	// Delete all existing basic auth providers (alternatively, we could not try to register one if there is
	// already an existing one, but that would get us into trouble when we change anything about the logic/config).
	// We have now stopped storing basic auth providers, but we might still hit this if we're upgrading from an
	// older version.
	typ := basicAuthProvider.TypeName
	existingBasicAuthProviders := registry.GetProviders(nil, &typ)
	for _, provider := range existingBasicAuthProviders {
		if err := registry.DeleteProvider(ctx, provider.ID(), true, true); err != nil {
			log.Panicf("Could not delete existing basic auth provider %s: %v", provider.Name(), err)
		}
	}

	options := []authproviders.ProviderOption{
		authproviders.WithType(basicAuthProvider.TypeName),
		authproviders.WithName("Login with username/password"),
		authproviders.WithID(basicAuthProviderID),
		authproviders.WithEnabled(true),
		authproviders.WithActive(true),
		authproviders.WithRoleMapper(mapper.AlwaysAdminRoleMapper()),
		authproviders.DoNotStore(),
	}

	// For managed services, we do not want to show the basic auth provider for login purposes. The default auth
	// in that context will be the sso.redhat.com auth provider.
	if env.ManagedCentral.BooleanSetting() {
		options = append(options, authproviders.WithVisibility(storage.Traits_HIDDEN))
	}

	provider, err := registry.CreateProvider(basicAuthProvider.ContextWithBasicAuthManager(ctx, mgr), options...)
	if err != nil {
		log.Panicf("Could not set up basic auth provider: %v", err)
	}
	return provider
}

// IdentityExtractorOrPanic creates and returns the identity extractor for basic authentication.
func IdentityExtractorOrPanic(store roleDatastore.DataStore, mgr *basicAuthn.Manager, authProvider authproviders.Provider) authn.IdentityExtractor {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	adminRole, found, err := store.GetRole(ctx, accesscontrol.Admin)
	if err != nil || !found || adminRole == nil {
		log.Panic("Could not look up admin role")
	}

	extractor, err := basicAuthn.NewExtractor(mgr, authProvider)
	if err != nil {
		log.Panicf("Could not create identity extractor for basic auth: %v", err)
	}

	wh := &watchHandler{
		manager: mgr,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}

	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), htpasswdDir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)

	return extractor
}

// IsLocalAdmin checks if the given identity is a local administrator (basic auth user, or token derived from that).
func IsLocalAdmin(id authn.Identity) bool {
	if id == nil {
		return false
	}

	if basicAuthn.IsBasicIdentity(id) {
		return true
	}
	provider := id.ExternalAuthProvider()
	if provider == nil {
		return false
	}
	return provider.Type() == basicAuthProvider.TypeName && provider.ID() == basicAuthProviderID
}
