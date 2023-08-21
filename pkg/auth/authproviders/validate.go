package authproviders

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	leeway = -10 * time.Second
)

// validateTokenProviderUpdate validates whether the claims are still valid for the given provider.
// In case the provider was updated _after_ the claims have been issued, they can be seen as invalid.
// This is due to e.g. changes in role mappings or claim structures.
func validateTokenProviderUpdate(provider *storage.AuthProvider, claims *tokens.Claims) error {
	lastProviderUpdate := timestamp.FromProtobuf(provider.GetLastUpdated())

	// Adding a leeway of 10 seconds to the time comparison.
	// Reasoning for this is that during the _first_ time an auth provider is used, the auth provider is marked as
	// `active` _after_ the token has been issued. This leads to a small different between `IssuedAt` and `LastUpdated`.
	// While the login attempt could just be retried, we'll add a leeway of 10 seconds to the `LastUpdated` value.
	if claims.IssuedAt.Time().Before(lastProviderUpdate.GoTime().Add(leeway)) {
		return errors.Errorf("token was issued at %s, but the provider was updated afterwards %s",
			claims.IssuedAt.Time().String(), lastProviderUpdate.GoTime().String())
	}

	return nil
}
