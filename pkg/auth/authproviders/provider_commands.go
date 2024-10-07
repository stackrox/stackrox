package authproviders

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/protocompat"
)

// Commands that providers can execute.
// So that we can keep Provider opaque, and to decouple store operations from the registry,
// These commands are temporarily applied as options.
/////////////////////////////////////////////////////

// DefaultAddToStore adds the providers stored data to the input store.
func DefaultAddToStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.doNotStore {
			return noOpRevert, nil
		}
		revert := getRevertLastUpdated(pr)
		if pr.storedInfo.LastUpdated == nil {
			pr.storedInfo.LastUpdated = protocompat.TimestampNow()
		}
		revert = getRemoveProvider(ctx, pr, store, revert)
		return revert, store.AddAuthProvider(ctx, pr.storedInfo)
	}
}

func getRevertLastUpdated(pr *providerImpl) RevertOption {
	oldLastUpdated := pr.storedInfo.GetLastUpdated()
	return func(pr *providerImpl) error {
		if pr.storedInfo == nil {
			return noStoredInfoErrox
		}
		pr.storedInfo.LastUpdated = oldLastUpdated
		return nil
	}
}

func getRemoveProvider(ctx context.Context, provider *providerImpl, store Store, revert RevertOption) RevertOption {
	providerID := provider.storedInfo.GetId()
	removeProviderAction := func(pr *providerImpl) error {
		return store.RemoveAuthProvider(ctx, providerID, true)
	}
	return composeRevertOptions(removeProviderAction, revert)
}

// UpdateStore updates the stored value for the provider in the input store.
func UpdateStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		if pr.doNotStore {
			return noOpRevert, nil
		}
		revert := getRevertLastUpdated(pr)
		pr.storedInfo.LastUpdated = protocompat.TimestampNow()
		revert = getRevertUpdateProvider(ctx, pr, store, revert)
		return revert, store.UpdateAuthProvider(ctx, pr.storedInfo)
	}
}

func getRevertUpdateProvider(ctx context.Context, provider *providerImpl, store Store, revert RevertOption) RevertOption {
	oldProviderID := provider.storedInfo.GetId()
	oldProvider, found, err := store.GetAuthProvider(ctx, oldProviderID)
	if err != nil {
		return noOpRevert
	}
	revertUpdateAction := func(_ *providerImpl) error {
		if !found {
			return store.RemoveAuthProvider(ctx, oldProviderID, true)
		}
		return store.UpdateAuthProvider(ctx, oldProvider)
	}
	return composeRevertOptions(revertUpdateAction, revert)
}

// DeleteFromStore removes the providers stored data from the input store.
func DeleteFromStore(ctx context.Context, store Store, providerID string, force bool) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		revert := getRevertDeleteProvider(ctx, store, providerID)
		err := store.RemoveAuthProvider(ctx, providerID, force)
		if err != nil {
			// If it's a type we don't want to store, then we're okay with it not existing.
			// We do this in case it was stored in the DB in a previous version.
			// The revert action will anyway restore what was in the DB prior to the removal,
			// if there was anything in DB.
			if pr.doNotStore && dberrors.IsNotFound(err) {
				return revert, nil
			}
			return revert, err
		}

		revert = getRestoreStoredInfo(pr, revert)
		// a deleted provider should no longer be accessible, but it's still cached as a token source so mark it as
		// no longer valid
		pr.storedInfo = &storage.AuthProvider{
			Id:      pr.storedInfo.GetId(),
			Enabled: false,
		}
		return revert, nil
	}
}

func getRevertDeleteProvider(ctx context.Context, store Store, providerID string) RevertOption {
	oldProvider, found, err := store.GetAuthProvider(ctx, providerID)
	if err != nil {
		return noOpRevert
	}
	if !found {
		// No DB data to re-create.
		return noOpRevert
	}
	return func(_ *providerImpl) error {
		return store.AddAuthProvider(ctx, oldProvider)
	}
}

func getRestoreStoredInfo(provider *providerImpl, revert RevertOption) RevertOption {
	oldStoredInfo := provider.storedInfo
	restoreStoredInfoAction := func(pr *providerImpl) error {
		pr.storedInfo = oldStoredInfo
		return nil
	}
	return composeRevertOptions(restoreStoredInfoAction, revert)
}

// UnregisterSource unregisters the token source from the source factory
func UnregisterSource(factory tokens.IssuerFactory) ProviderOption {
	return func(pr *providerImpl) (RevertOption, error) {
		err := factory.UnregisterSource(pr)
		// both DeleteFromStore and UnregisterSource mutate external stores, so regardless of order the second one
		// can't return err and fail the change.
		if err != nil {
			log.Warnf("Unable to unregister token source: %v", err)
		}
		return noOpRevert, nil
	}
}
