package authproviders

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	internalUpdateProviderCtx = sac.WithAllAccess(context.Background())

	noStoredInfoErrox = errox.InvariantViolation.CausedBy("no storage data for auth provider")
)

// RevertOption is a function that modifies a providerImpl, undoing the work of a ProviderOption.
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
		revert := getRevertBackendFactoryFunc(pr)
		pr.backendFactory = factory

		revert = getRevertBackendFunc(pr, revert)
		backend, err := factory.CreateBackend(ctx, pr.storedInfo.GetId(), AllUIEndpoints(pr.storedInfo), pr.storedInfo.GetConfig(), nil)
		if err != nil {
			return revert, errors.Wrapf(err, "failed to create auth provider of type %s", pr.storedInfo.GetType())
		}
		pr.backend = backend

		revert = getRevertProviderConfigFunc(pr, revert)
		pr.storedInfo.Config = backend.Config()
		return revert, nil
	}
}

func getRevertBackendFactoryFunc(provider *providerImpl) RevertOption {
	oldBackendFactory := provider.backendFactory
	return func(pr *providerImpl) error {
		pr.backendFactory = oldBackendFactory
		return nil
	}
}

func getRevertBackendFunc(provider *providerImpl, revert RevertOption) RevertOption {
	oldBackend := provider.Backend()
	revertBackendAction := func(pr *providerImpl) error {
		backendID := pr.storedInfo.GetId()
		pr.backendFactory.CleanupBackend(backendID)
		pr.backend = oldBackend
		return nil
	}
	return composeRevertOptions(revertBackendAction, revert)
}

func getRevertProviderConfigFunc(provider *providerImpl, revert RevertOption) RevertOption {
	oldConfig := provider.storedInfo.GetConfig()
	revertConfigAction := func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErr
		}
		pr.storedInfo.Config = oldConfig
		return nil
	}
	return composeRevertOptions(revertConfigAction, revert)
}

// DoNotStore indicates that this provider should not be stored.
func DoNotStore() ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		revert := getRevertDoNotStoreFunc(pr)
		pr.doNotStore = true
		return revert, nil
	}
}

func getRevertDoNotStoreFunc(provider *providerImpl) RevertOption {
	oldDoNotStore := provider.doNotStore
	return func(pr *providerImpl) error {
		pr.doNotStore = oldDoNotStore
		return nil
	}
}

// WithRoleMapper adds a role mapper to the provider.
func WithRoleMapper(roleMapper permissions.RoleMapper) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		revert := getRevertRoleMapperFunc(pr)
		pr.roleMapper = roleMapper
		return revert, nil
	}
}

func getRevertRoleMapperFunc(provider *providerImpl) RevertOption {
	oldRoleMapper := provider.RoleMapper()
	return func(pr *providerImpl) error {
		pr.roleMapper = oldRoleMapper
		return nil
	}
}

// WithStorageView sets the values in the store auth provider from the input value.
func WithStorageView(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		revert := getRevertStorageViewFunc(pr)
		pr.storedInfo = stored.CloneVT()
		return revert, nil
	}
}

func getRevertStorageViewFunc(pr *providerImpl) RevertOption {
	oldStoredInfo := pr.storedInfo.CloneVT()
	return func(pr *providerImpl) error {
		pr.storedInfo = oldStoredInfo
		return nil
	}
}

// WithID sets the id for the provider to the input value.
func WithID(id string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, noStoredInfoErrox
		}
		revert := getRevertIDFunc(pr, noStoredInfoErrox)
		pr.storedInfo.Id = id
		return revert, nil
	}
}

func getRevertIDFunc(provider *providerImpl, noStoredInfoErr error) RevertOption {
	oldID := provider.storedInfo.GetId()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErr
		}
		pr.storedInfo.Id = oldID
		return nil
	}
}

// WithType sets the type for the provider.
func WithType(typ string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, noStoredInfoErrox
		}
		revert := getRevertTypeFunc(pr)
		pr.storedInfo.Type = typ
		return revert, nil
	}
}

func getRevertTypeFunc(provider *providerImpl) RevertOption {
	oldType := provider.storedInfo.GetType()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		pr.storedInfo.Type = oldType
		return nil
	}
}

// WithName sets the name for the provider.
func WithName(name string) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, noStoredInfoErrox
		}
		revert := getRevertNameFunc(pr)
		pr.storedInfo.Name = name
		return revert, nil
	}
}

func getRevertNameFunc(provider *providerImpl) RevertOption {
	oldName := provider.storedInfo.GetName()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		pr.storedInfo.Name = oldName
		return nil
	}
}

// WithEnabled sets the enabled flag for the provider.
func WithEnabled(enabled bool) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, noStoredInfoErrox
		}
		revert := getRevertEnabledFunc(pr)
		pr.storedInfo.Enabled = enabled
		return revert, nil
	}
}

func getRevertEnabledFunc(provider *providerImpl) RevertOption {
	oldEnabled := provider.storedInfo.GetEnabled()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		pr.storedInfo.Enabled = oldEnabled
		return nil
	}
}

// WithValidateCallback adds a callback to validate the auth provider.
func WithValidateCallback(store Store) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		revert := getRevertValidateCallbackFunc(pr)
		pr.validateCallback = func() error {
			return pr.ApplyOptions(WithActive(true), UpdateStore(internalUpdateProviderCtx, store))
		}
		return revert, nil
	}
}

func getRevertValidateCallbackFunc(provider *providerImpl) RevertOption {
	oldValidateCallback := provider.validateCallback
	return func(pr *providerImpl) error {
		pr.validateCallback = oldValidateCallback
		return nil
	}
}

// WithActive sets the active flag for the provider.
func WithActive(active bool) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, errox.InvariantViolation.CausedBy("no storage data for auth provider")
		}
		revert := getRevertActiveFunc(pr)
		pr.storedInfo.Validated = active
		pr.storedInfo.Active = active
		return revert, nil
	}
}

func getRevertActiveFunc(provider *providerImpl) RevertOption {
	oldActive := provider.storedInfo.GetActive()
	oldValidated := provider.storedInfo.GetValidated()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		pr.storedInfo.Active = oldActive
		pr.storedInfo.Validated = oldValidated
		return nil
	}
}

// WithAttributeVerifier adds an attribute verifier to the provider based on the list of
// required attributes from the provided auth provider instance.
func WithAttributeVerifier(stored *storage.AuthProvider) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if stored.GetRequiredAttributes() == nil {
			return noOpRevert, nil
		}
		revert := getRevertAttributeVerifierFunc(pr)
		pr.attributeVerifier = user.NewRequiredAttributesVerifier(stored.GetRequiredAttributes())
		return revert, nil
	}
}

func getRevertAttributeVerifierFunc(provider *providerImpl) RevertOption {
	oldAttributeVerifier := provider.attributeVerifier
	return func(pr *providerImpl) error {
		pr.attributeVerifier = oldAttributeVerifier
		return nil
	}
}

// WithVisibility sets the visibility for the auth provider.
func WithVisibility(visibility storage.Traits_Visibility) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.storedInfo == nil {
			return noOpRevert, noStoredInfoErrox
		}
		revert := getRevertVisibilityFunc(pr)
		if pr.storedInfo.GetTraits() != nil {
			pr.storedInfo.Traits.Visibility = visibility
		} else {
			pr.storedInfo.Traits = &storage.Traits{Visibility: visibility}
		}
		return revert, nil
	}
}

func getRevertVisibilityFunc(provider *providerImpl) RevertOption {
	oldTraits := provider.storedInfo.GetTraits()
	oldVisibility := oldTraits.GetVisibility()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		if oldTraits == nil {
			pr.storedInfo.Traits = nil
		} else {
			pr.storedInfo.Traits.Visibility = oldVisibility
		}
		return nil
	}
}

func noOpRevert(_ *providerImpl) error { return nil }

func composeRevertOptions(revertActions ...RevertOption) RevertOption {
	if len(revertActions) == 0 {
		return noOpRevert
	}
	if len(revertActions) == 1 {
		return revertActions[0]
	}
	return func(pr *providerImpl) error {
		var err *multierror.Error
		for _, revertAction := range revertActions {
			actionErr := revertAction(pr)
			if actionErr != nil {
				err = multierror.Append(err, actionErr)
			}
		}
		return err.ErrorOrNil()
	}
}
