package authproviders

import (
	"context"
	"net/http"
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

	CreateAuthProvider(ctx context.Context, typ, name string, uiEndpoints []string, enabled bool, validated bool, config map[string]string) (Provider, error)
	UpdateAuthProvider(ctx context.Context, id string, name *string, enabled *bool) (Provider, error)
	GetAuthProvider(ctx context.Context, id string) Provider
	GetAuthProviders(ctx context.Context, name, typ *string) []Provider
	DeleteAuthProvider(ctx context.Context, id string) error
	ExchangeToken(ctx context.Context, externalToken string, typ string, state string) (string, string, error)

	// RegisterBackendFactory registers the given factory (creator) under the specified type. The creation of the
	// factory is not delayed; the reason this function does not receive a factory instance directly is only to allow
	// passing the URL prefix.
	RegisterBackendFactory(typ string, factoryCreator BackendFactoryCreator) error

	// HasUsableProviders returns whether there are any usable (i.e., enabled and validated) auth providers.
	HasUsableProviders() bool
}
