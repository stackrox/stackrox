package authproviders

import (
	"context"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
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
		if pr.storedInfo.LastUpdated == nil {
			pr.storedInfo.LastUpdated = timestamp.TimestampNow()
		}
		return store.AddAuthProvider(ctx, pr.storedInfo)
	}
}

// UpdateStore updates the stored value for the provider in the input store.
func UpdateStore(ctx context.Context, store Store) ProviderOption {
	return func(pr *providerImpl) error {
		if pr.doNotStore {
			return nil
		}
		pr.storedInfo.LastUpdated = timestamp.TimestampNow()
		return store.UpdateAuthProvider(ctx, pr.storedInfo)
	}
}

// DeleteFromStore removes the providers stored data from the input store.
func DeleteFromStore(ctx context.Context, store Store, providerID string, force bool) ProviderOption {
	return func(pr *providerImpl) error {
		err := store.RemoveAuthProvider(ctx, providerID, force)
		if err != nil {
			// If it's a type we don't want to store, then we're okay with it not existing.
			// We do this in case it was stored in the DB in a previous version.
			if pr.doNotStore && dberrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		// a deleted provider should no longer be accessible, but it's still cached as a token source so mark it as
		// no longer valid
		pr.storedInfo = &storage.AuthProvider{
			Id:      pr.storedInfo.GetId(),
			Enabled: false,
		}
		return nil
	}
}

// UnregisterSource unregisters the token source from the source factory
func UnregisterSource(factory tokens.IssuerFactory) ProviderOption {
	return func(pr *providerImpl) error {
		err := factory.UnregisterSource(pr)
		// both DeleteFromStore and UnregisterSource mutate external stores, so regardless of order the second one
		// can't return err and fail the change.
		if err != nil {
			log.Warnf("Unable to unregister token source: %v", err)
		}
		return nil
	}
}
