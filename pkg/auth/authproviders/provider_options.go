package authproviders

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// ProviderOption is a function that modifies a providerImpl.
// Do not use Provider functions in Options, as this will try to RLock inside of a Lock and deadlock.
// You can assume that the provider is locked for the duration of the option's execution.
type ProviderOption func(*providerImpl) error

// Options for building and updating.
/////////////////////////////////////

// WithBackendFromFactory adds a backend from the factory to the provider.
func WithBackendFromFactory(factory BackendFactory) ProviderOption {
	return func(pr *providerImpl) error {
		backend, effectiveConfig, err := factory.CreateBackend(context.Background(), pr.storedInfo.Id, AllUIEndpoints(&pr.storedInfo), pr.storedInfo.Config)
		if err != nil {
			return errors.Wrapf(err, "failed to create auth provider of type %s", pr.storedInfo.Type)
		}

		pr.backend = backend
		pr.storedInfo.Config = effectiveConfig
		return nil
	}
}

// DoNotStore indicates that this provider should not be stored.
func DoNotStore() ProviderOption {
	return func(pr *providerImpl) error {
		pr.doNotStore = true
		return nil
	}
}

// WithRoleMapper adds a role mapper to the provider.
func WithRoleMapper(roleMapper permissions.RoleMapper) ProviderOption {
	return func(pr *providerImpl) error {
		pr.roleMapper = roleMapper
		return nil
	}
}

// WithStorageView sets the values in the store auth provider from the input value.
func WithStorageView(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo = *stored
		return nil
	}
}

// WithID sets the id for the provider to the input value.
func WithID(id string) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Id = id
		return nil
	}
}

// WithType sets the type for the provider.
func WithType(typ string) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Type = typ
		return nil
	}
}

// WithName sets the name for the provider.
func WithName(name string) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Name = name
		return nil
	}
}

// WithEnabled sets the enabled flag for the provider.
func WithEnabled(enabled bool) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Enabled = enabled
		return nil
	}
}

// WithValidated sets the validated flag for the provider.
func WithValidated(validated bool) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Validated = validated
		return nil
	}
}

// WithConfig sets the config for the provider.
func WithConfig(config map[string]string) ProviderOption {
	return func(pr *providerImpl) error {
		pr.storedInfo.Config = config
		return nil
	}
}

// WithSuccessCallback sets the option to execute when OnSuccess is called.
func WithSuccessCallback(onSuccess ProviderOption) ProviderOption {
	return func(pr *providerImpl) error {
		pr.onSuccess = onSuccess
		return nil
	}
}
