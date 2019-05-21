package authproviders

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// An Provider is an authenticator which is based on an external service, like auth0.
type Provider interface {
	tokens.Source

	Name() string
	Type() string
	Enabled() bool

	// StorageView returns a description of the authentication provider in protobuf format.
	StorageView() *storage.AuthProvider
	Backend() Backend
	RoleMapper() permissions.RoleMapper
	Issuer() tokens.Issuer

	OnSuccess() error
	applyOptions(options ...ProviderOption) error
}

// NewProvider creates a new provider with the input options.
func NewProvider(options ...ProviderOption) (Provider, error) {
	provider := &providerImpl{}
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
	if provider.storedInfo.Id == "" {
		return fmt.Errorf("auth providers must have an id")
	}
	if provider.storedInfo.Name == "" {
		return fmt.Errorf("auth providers must have a name")
	}
	return nil
}
