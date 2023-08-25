package authproviders

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	internalUpdateProviderCtx = sac.WithAllAccess(context.Background())
)

// ProviderOption is a function that modifies a providerImpl.
// Do not use Provider functions in Options, as this will try to RLock inside of a Lock and deadlock.
// You can assume that the provider is locked for the duration of the option's execution.
type ProviderOption func(*providerImpl) error

// Options for building and updating.
/////////////////////////////////////

// WithBackendFromFactory adds a backend from the factory to the provider.
func WithBackendFromFactory(ctx context.Context, factory BackendFactory) ProviderOption {
	return func(pr *providerImpl) error {
		pr.backendFactory = factory

		backend, err := factory.CreateBackend(ctx, pr.storedInfo.GetId(), AllUIEndpoints(pr.storedInfo), pr.storedInfo.GetConfig(), nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create auth provider of type %s", pr.storedInfo.GetType())
		}

		pr.backend = backend
		pr.storedInfo.Config = backend.Config()
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
		pr.storedInfo = stored.Clone()
		return nil
	}
}

// WithID sets the id for the provider to the input value.
func WithID(id string) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Id = id
		return nil
	}
}

// WithType sets the type for the provider.
func WithType(typ string) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Type = typ
		return nil
	}
}

// WithName sets the name for the provider.
func WithName(name string) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Name = name
		return nil
	}
}

// WithEnabled sets the enabled flag for the provider.
func WithEnabled(enabled bool) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Enabled = enabled
		return nil
	}
}

// WithValidateCallback adds a callback to validate the auth provider.
func WithValidateCallback(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		pr.validateCallback = func() error {
			return pr.ApplyOptions(WithActive(true), UpdateStore(internalUpdateProviderCtx, store))
		}
		return nil
	}
}

// WithActive sets the active flag for the provider.
func WithActive(active bool) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Validated = active
		pr.storedInfo.Active = active
		return nil
	}
}

// WithConfig sets the config for the provider.
func WithConfig(config map[string]string) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		pr.storedInfo.Config = config
		return nil
	}
}

// WithAttributeVerifier adds an attribute verifier to the provider based on the list of
// required attributes from the provided auth provider instance.
func WithAttributeVerifier(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) error {
		if stored.GetRequiredAttributes() == nil {
			return nil
		}
		pr.attributeVerifier = user.NewRequiredAttributesVerifier(stored.GetRequiredAttributes())
		return nil
	}
}

// WithVisibility sets the visibility for the auth provider.
func WithVisibility(visibility storage.Traits_Visibility) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		if pr.storedInfo.GetTraits() != nil {
			pr.storedInfo.Traits.Visibility = visibility
		} else {
			pr.storedInfo.Traits = &storage.Traits{Visibility: visibility}
		}
		return nil
	}
}
