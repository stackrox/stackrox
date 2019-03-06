package userpass

import (
	"context"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	basicAuthProvider "github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/grpc/authn"
	basicAuthn "github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	htpasswdFile = "/run/secrets/stackrox.io/htpasswd/htpasswd"
)

var (
	log = logging.LoggerForModule()
)

// RegisterAuthProviderOrPanic sets up basic authentication with the builtin htpasswd file. It panics if the basic auth
// feature is not enabled, or if it is called twice on the same registry.
func RegisterAuthProviderOrPanic(registry authproviders.Registry) {
	err := registry.RegisterBackendFactory(basicAuthProvider.TypeName, basicAuthProvider.NewFactory)
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
		if err := registry.DeleteProvider(provider.ID()); err != nil {
			log.Panicf("Could not delete existing basic auth provider %s: %v", provider.Name(), err)
		}
	}

	config := map[string]string{
		"htpasswd_file": htpasswdFile,
	}
	options := []authproviders.ProviderOption{
		authproviders.WithType(basicAuthProvider.TypeName),
		authproviders.WithName("Login with username/password"),
		authproviders.WithEnabled(true),
		authproviders.WithValidated(true),
		authproviders.WithConfig(config),
		authproviders.WithRoleMapper(mapper.AlwaysAdminRoleMapper()),
		authproviders.DoNotStore(),
	}
	_, err = registry.CreateProvider(context.Background(), options...)
	if err != nil {
		log.Panicf("Could not set up basic auth provider: %v", err)
	}
}

// IdentityExtractorOrPanic creates and returns the identity extractor for basic authentication.
func IdentityExtractorOrPanic() authn.IdentityExtractor {
	adminRole := role.DefaultRolesByName[role.Admin]
	if adminRole == nil {
		log.Panic("Could not look up admin role")
	}
	extractor, err := basicAuthn.NewExtractor(htpasswdFile, adminRole)
	if err != nil {
		log.Panicf("Could not create identity extractor for basic auth: %v", err)
	}
	return extractor
}
