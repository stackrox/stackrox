package authproviders

import (
	"context"

	"github.com/stackrox/rox/pkg/dberrors"
)

// Commands that providers can execute.
// So that we can keep Provider opaque, and to decouple store operations from the registry,
// These commands are temporarily applied as options.
/////////////////////////////////////////////////////

// DefaultAddToStore adds the providers stored data to the input store.
func DefaultAddToStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return nil
		}
		return store.AddAuthProvider(ctx, &pr.storedInfo)
	}
}

// UpdateStore updates the stored value for the provider in the input store.
func UpdateStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return nil
		}
		return store.UpdateAuthProvider(ctx, &pr.storedInfo)
	}
}

// DeleteFromStore removes the providers stored data from the input store.
func DeleteFromStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) error {
		err := store.RemoveAuthProvider(ctx, pr.storedInfo.Id)
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
