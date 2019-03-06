package authproviders

import (
	"fmt"

	"github.com/stackrox/rox/pkg/dberrors"
)

// Commands that providers can execute.
// So that we can keep Provider opaque, and to decouple store operations from the registry,
// These commands are temporarily applied as options.
/////////////////////////////////////////////////////

// DefaultAddToStore adds the providers stored data to the input store.
func DefaultAddToStore(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return nil
		}
		return store.AddAuthProvider(&pr.storedInfo)
	}
}

// UpdateStore updates the stored value for the provider in the input store.
func UpdateStore(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return nil
		}
		return store.UpdateAuthProvider(&pr.storedInfo)
	}
}

// RecordSuccess sets the 'validated' flag both for the provider and in the store.
func RecordSuccess(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return fmt.Errorf("provider '%s' is not stored, cannot record success", pr.storedInfo.Name)
		}
		if !pr.storedInfo.Enabled {
			return fmt.Errorf("cannot record success for disabled auth provider: %s", pr.storedInfo.Id)
		}

		err := store.RecordAuthSuccess(pr.storedInfo.Id)
		if err != nil {
			return err
		}
		pr.storedInfo.Validated = true
		return nil
	}
}

// DeleteFromStore removes the providers stored data from the input store.
func DeleteFromStore(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		err := store.RemoveAuthProvider(pr.storedInfo.Id)
		if err != nil {
			// If it's a type we don't want to store, then we're okay with it not existing.
			// We do this in case it was stored in the DB in a previous version.
			if pr.doNotStore && dberrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		return nil
	}
}
