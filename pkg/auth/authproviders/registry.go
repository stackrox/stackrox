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

	CreateProvider(ctx context.Context, options ...ProviderOption) (Provider, error)
	UpdateProvider(id string, options ...ProviderOption) (Provider, error)
	GetProvider(id string) Provider
	GetProviders(name, typ *string) []Provider
	DeleteProvider(id string) error
	ExchangeToken(ctx context.Context, externalToken string, typ string, state string) (string, string, error)

	// RegisterBackendFactory registers the given factory (creator) under the specified type. The creation of the
	// factory is not delayed; the reason this function does not receive a factory instance directly is only to allow
	// passing the URL prefix.
	RegisterBackendFactory(typ string, factoryCreator BackendFactoryCreator) error
}
