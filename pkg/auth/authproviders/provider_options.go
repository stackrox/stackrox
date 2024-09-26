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

// RevertOption is a function that modifies a providerImpl.
// Do not use Provider functions in Options, as this will try to RLock inside of a Lock and deadlock.
// You can assume that the provider is locked for the duration of the option's execution.
type RevertOption func(*providerImpl) error

// ProviderOption is a function that modifies a providerImpl.
// Do not use Provider functions in Options, as this will try to RLock inside of a Lock and deadlock.
// You can assume that the provider is locked for the duration of the option's execution.
type ProviderOption func(*providerImpl) (RevertOption, error)

// Options for building and updating.
/////////////////////////////////////

// WithBackendFromFactory adds a backend from the factory to the provider.
func WithBackendFromFactory(ctx context.Context, factory BackendFactory) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldBackendFactory := pr.backendFactory
		oldBackend := pr.backend
		oldConfig := pr.storedInfo.GetConfig()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo != nil {
				pr.storedInfo.Config = oldConfig
			}
			backendID := pr.storedInfo.GetId()
			pr.backend = oldBackend
			err := pr.backendFactory.CleanupBackend(backendID)
			pr.backendFactory = oldBackendFactory
			return err
		}

		pr.backendFactory = factory

		backend, err := factory.CreateBackend(ctx, pr.storedInfo.GetId(), AllUIEndpoints(pr.storedInfo), pr.storedInfo.GetConfig(), nil)
		if err != nil {
			return revert, errors.Wrapf(err, "failed to create auth provider of type %s", pr.storedInfo.GetType())
		}

		pr.backend = backend
		pr.storedInfo.Config = backend.Config()
		return revert, nil
	}
}

// DoNotStore indicates that this provider should not be stored.
func DoNotStore() ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldDoNotStore := pr.doNotStore
		revert := func(pr *providerImpl) error {
			pr.doNotStore = oldDoNotStore
			return nil
		}
		pr.doNotStore = true
		return revert, nil
	}
}

// WithRoleMapper adds a role mapper to the provider.
func WithRoleMapper(roleMapper permissions.RoleMapper) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldRoleMapper := pr.roleMapper
		revert := func(pr *providerImpl) error {
			pr.roleMapper = oldRoleMapper
			return nil
		}
		pr.roleMapper = roleMapper
		return revert, nil
	}
}

// WithStorageView sets the values in the store auth provider from the input value.
func WithStorageView(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldStoredInfo := pr.storedInfo
		revert := func(pr *providerImpl) error {
			pr.storedInfo = oldStoredInfo
			return nil
		}
		pr.storedInfo = stored.CloneVT()
		return revert, nil
	}
}

// WithID sets the id for the provider to the input value.
func WithID(id string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldId := pr.storedInfo.GetId()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			pr.storedInfo.Id = oldId
			return nil
		}
		pr.storedInfo.Id = id
		return revert, nil
	}
}

// WithType sets the type for the provider.
func WithType(typ string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldType := pr.storedInfo.GetType()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			pr.storedInfo.Type = oldType
			return nil
		}
		pr.storedInfo.Type = typ
		return revert, nil
	}
}

// WithName sets the name for the provider.
func WithName(name string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldName := pr.storedInfo.GetName()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			pr.storedInfo.Name = oldName
			return nil
		}
		pr.storedInfo.Name = name
		return revert, nil
	}
}

// WithEnabled sets the enabled flag for the provider.
func WithEnabled(enabled bool) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldEnabled := pr.storedInfo.GetEnabled()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			pr.storedInfo.Enabled = oldEnabled
			return nil
		}
		pr.storedInfo.Enabled = enabled
		return revert, nil
	}
}

// WithValidateCallback adds a callback to validate the auth provider.
func WithValidateCallback(store Store) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldValidateCallback := pr.validateCallback
		revert := func(pr *providerImpl) error {
			pr.validateCallback = oldValidateCallback
			return nil
		}
		pr.validateCallback = func() error {
			return pr.ApplyOptions(WithActive(true), UpdateStore(internalUpdateProviderCtx, store))
		}
		return revert, nil
	}
}

// WithActive sets the active flag for the provider.
func WithActive(active bool) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldValidated := pr.storedInfo.GetValidated()
		oldActive := pr.storedInfo.GetActive()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			pr.storedInfo.Active = oldActive
			pr.storedInfo.Validated = oldValidated
			return nil
		}
		pr.storedInfo.Validated = active
		pr.storedInfo.Active = active
		return revert, nil
	}
}

// WithAttributeVerifier adds an attribute verifier to the provider based on the list of
// required attributes from the provided auth provider instance.
func WithAttributeVerifier(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		oldAttributeVerifier := pr.attributeVerifier
		revert := func(pr *providerImpl) error {
			pr.attributeVerifier = oldAttributeVerifier
			return nil
		}
		if stored.GetRequiredAttributes() == nil {
			return noOpRevert, nil
		}
		pr.attributeVerifier = user.NewRequiredAttributesVerifier(stored.GetRequiredAttributes())
		return revert, nil
	}
}

// WithVisibility sets the visibility for the auth provider.
func WithVisibility(visibility storage.Traits_Visibility) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		oldTraits := pr.storedInfo.GetTraits()
		oldVisibility := oldTraits.GetVisibility()
		revert := func(pr *providerImpl) error {
			if pr.storedInfo == nil {
				return errox.InvariantViolation.CausedBy("no storage data for auth provider")
			}
			if oldTraits == nil {
				pr.storedInfo.Traits = nil
			} else {
				pr.storedInfo.Traits.Visibility = oldVisibility
			}
			return nil
		}
		if pr.storedInfo.GetTraits() != nil {
			pr.storedInfo.Traits.Visibility = visibility
		} else {
			pr.storedInfo.Traits = &storage.Traits{Visibility: visibility}
		}
		return revert, nil
	}
}

func noOpRevert(_ *providerImpl) error { return nil }
