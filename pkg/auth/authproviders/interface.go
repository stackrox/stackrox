package authproviders

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// An AuthProvider is an authenticator which is based on an external service, like auth0.
type AuthProvider interface {
	tokens.Source

	Name() string
	Type() string
	Enabled() bool
	Backend() AuthProviderBackend
	RoleMapper() permissions.RoleMapper

	// AsV1 returns a description of the authentication provider in protobuf format.
	AsV1() *v1.AuthProvider

	// RecordSuccess should be called the first time a user successfully logs in through an auth provider, to mark it as
	// validated. This is used to prevent a user from accidentally locking themselves out of the system by setting up a
	// misconfigured auth provider.
	RecordSuccess() error
}

// Registry stores information about registered authentication providers, as well as about the factories to create them.
// It also acts as an HTTP/1.1 handler, since most auth providers require some form of callback to the original webpage,
// which cannot be implemented as a GRPC function.
type Registry interface {
	http.Handler

	// URLPathPrefix returns the path prefix (including a trailing slash) for URLs handled by this registry.
	URLPathPrefix() string

	CreateAuthProvider(ctx context.Context, typ, name string, uiEndpoints []string, enabled bool, validated bool, config map[string]string) (AuthProvider, error)
	UpdateAuthProvider(ctx context.Context, id string, name *string, enabled *bool) (AuthProvider, error)
	GetAuthProvider(ctx context.Context, id string) AuthProvider
	GetAuthProviders(ctx context.Context, name, typ *string) []AuthProvider
	DeleteAuthProvider(ctx context.Context, id string) error
	ExchangeToken(ctx context.Context, externalToken string, typ string, state string) (string, string, error)

	// RegisterBackendFactory registers the given factory (creator) under the specified type. The creation of the
	// factory is not delayed; the reason this function does not receive a factory instance directly is only to allow
	// passing the URL prefix.
	RegisterBackendFactory(typ string, factoryCreator BackendFactoryCreator) error

	// HasUsableProviders returns whether there are any usable (i.e., enabled and validated) auth providers.
	HasUsableProviders() bool
}

// AuthProviderBackend is a backend for an authentication provider.
type AuthProviderBackend interface {
	// LoginURL returns a login URL with the given client state.
	LoginURL(clientState string, ri *requestinfo.RequestInfo) string
	// RefreshURL returns a refresh URL, if supported by the auth provider.
	RefreshURL() string

	// ProcessHTTPRequest dispatches HTTP/1.1 requests intended for this provider. If the request is a callback from
	// a login page, and the login was successful, the respective ExternalUserClaim is returned. If a non-login HTTP
	// call should be handled, a nil claim and error should be returned.
	ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error)
	// ExchangeToken is called to exchange an external token, referring to the auth provider, against a Rox-issued
	// token.
	ExchangeToken(ctx context.Context, externalToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error)
}
