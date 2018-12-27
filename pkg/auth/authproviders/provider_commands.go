package authproviders

import (
	"fmt"
)

// Commands that providers can execute.
// So that we can keep Provider opaque, and to decouple store operations from the registry,
// These commands are temporarily applied as options.
/////////////////////////////////////////////////////

// AddToStore adds the providers stored data to the input store.
func AddToStore(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		return store.AddAuthProvider(&pr.storedInfo)
	}
}

// UpdateStore updates the stored value for the provider in the input store.
func UpdateStore(store Store) ProviderOption {
	return func(pr *providerImpl) error {
		return store.UpdateAuthProvider(&pr.storedInfo)
	}
}

// RecordSuccess sets the 'validated' flag both for the provider and in the store.
func RecordSuccess(store Store) ProviderOption {
	return func(pr *providerImpl) error {
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
		return store.RemoveAuthProvider(pr.storedInfo.Id)
	}
}
