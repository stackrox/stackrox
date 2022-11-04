package authproviders

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
)

// An Provider is an authenticator which is based on an external service, like auth0.
type Provider interface {
	tokens.Source

	Name() string
	Type() string

	// Enabled returns whether this auth provider is enabled. Note that an enabled auth provider can have a `nil`
	// Backend (if there were errors instantiating it) and vice versa.
	Enabled() bool

	MergeConfigInto(newCfg map[string]string) map[string]string

	// StorageView returns a description of the authentication provider in protobuf format.
	// Any secrets are redacted.
	StorageView() *storage.AuthProvider

	BackendFactory() BackendFactory

	// Backend returns the backend of this auth provider, if one exists. The result might be nil if there was an error
	// instantiating this backend. Note that whether a `nil` or non-`nil` value is returned here is entirely independent
	// of what Enabled() returns.
	Backend() Backend

	GetOrCreateBackend(ctx context.Context) (Backend, error)

	RoleMapper() permissions.RoleMapper
	Issuer() tokens.Issuer

	// AttributeVerifier is optional. If it is set, external user attributes MUST be verified
	// with the set user.AttributeVerifier. Otherwise, it would lead to authenticating principals that should be denied
	// authentication.
	AttributeVerifier() user.AttributeVerifier

	ApplyOptions(options ...ProviderOption) error
	Active() bool
	MarkAsActive() error
}

// NewProvider creates a new provider with the input options.
func NewProvider(options ...ProviderOption) (Provider, error) {
	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	if err := applyOptions(provider, options...); err != nil {
		return nil, err
	}
	if err := validateProvider(provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// Input provider must be locked when run.
func applyOptions(provider *providerImpl, options ...ProviderOption) error {
	for _, option := range options {
		if err := option(provider); err != nil {
			return err
		}
	}
	return nil
}

// Input provider must be locked when run.
func validateProvider(provider *providerImpl) error {
	if provider.storedInfo.GetId() == "" {
		return errors.New("auth providers must have an id")
	}
	if provider.storedInfo.GetName() == "" {
		return errors.New("auth providers must have a name")
	}
	return nil
}
