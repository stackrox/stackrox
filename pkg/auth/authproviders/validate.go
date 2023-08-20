package authproviders

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/timestamp"
)

// validateTokenProviderUpdate validates whether the claims are still valid for the given provider.
// In case the provider was updated _after_ the claims have been issued, they can be seen as invalid.
// This is due to e.g. changes in role mappings or claim structures.
func validateTokenProviderUpdate(provider *storage.AuthProvider, claims *tokens.Claims) error {
	lastProviderUpdate := timestamp.FromProtobuf(provider.GetLastUpdated())

	if claims.IssuedAt.Time().Before(lastProviderUpdate.GoTime()) {
		return errors.Errorf("token was issued at %s, but the provider was updated afterwards %s",
			claims.IssuedAt.Time().String(), lastProviderUpdate.GoTime().String())
	}

	return nil
}
