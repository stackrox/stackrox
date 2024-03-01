package authproviders

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

// Registry stores information about registered authentication providers, as well as about the factories to create them.
// It also acts as an HTTP/1.1 handler, since most auth providers require some form of callback to the original webpage,
// which cannot be implemented as a GRPC function.
type Registry interface {
	http.Handler

	// Init initializes the registry, including reading existing auth providers from the DB, if applicable.
	// This allows registering auth providers before reading the registry.
	Init() error

	// URLPathPrefix returns the path prefix (including a trailing slash) for URLs handled by this registry.
	URLPathPrefix() string

	GetProvider(id string) Provider
	GetProviders(name, typ *string) []Provider
	ResolveProvider(typ, state string) (Provider, error)

	ValidateProvider(ctx context.Context, options ...ProviderOption) error
	CreateProvider(ctx context.Context, options ...ProviderOption) (Provider, error)
	UpdateProvider(ctx context.Context, id string, options ...ProviderOption) (Provider, error)
	DeleteProvider(ctx context.Context, id string, force bool, ignoreActive bool) error

	// RegisterBackendFactory registers the given factory (creator) under the specified type. The creation of the
	// factory is not delayed; the reason this function does not receive a factory instance directly is only to allow
	// passing the URL prefix.
	RegisterBackendFactory(ctx context.Context, typ string, factoryCreator BackendFactoryCreator) error

	GetExternalUserClaim(ctx context.Context, externalToken, typ, state string) (*AuthResponse, string, error)
	IssueToken(ctx context.Context, provider Provider, authResponse *AuthResponse) (*tokens.TokenInfo, *http.Cookie, error)
	// GetBackendFactories returns all backend factories present.
	GetBackendFactories() map[string]BackendFactory
}
