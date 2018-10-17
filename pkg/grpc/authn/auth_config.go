package authn

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/contextutil"
)

var (
	// ErrNoContext is returned when we process a context, but can't find any Identity info.
	ErrNoContext = errors.New("no identity context found")
)

type authConfigurationContextKey struct{}

// AuthConfiguration provides information about how auth is configured (or not).
type AuthConfiguration struct {
	// ProviderConfigured indicates at least one provider is configured.
	ProviderConfigured bool
}

// FromAuthConfigurationContext retrieves information about authentication
// configuration from the given context. The context must have been passed
// through the interceptors provided by this package.
func FromAuthConfigurationContext(ctx context.Context) (AuthConfiguration, error) {
	val, ok := ctx.Value(authConfigurationContextKey{}).(AuthConfiguration)
	if !ok {
		return AuthConfiguration{}, ErrNoContext
	}
	return val, nil
}

// NewAuthConfigChecker returns a context updater that checks (and stores) if any authentication providers are configured.
func NewAuthConfigChecker(registry authproviders.Registry) contextutil.ContextUpdater {
	return authConfigChecker{registry: registry}.updateContext
}

type authConfigChecker struct {
	registry authproviders.Registry
}

func (c authConfigChecker) updateContext(ctx context.Context) (context.Context, error) {
	anyConfigured := false
	if c.registry != nil {
		anyConfigured = c.registry.HasUsableProviders()
	} else {
		// If providerAccessor is nil, there will never be any (configured) auth providers, so the configuration can be
		// considered final.
		anyConfigured = true
	}

	return context.WithValue(ctx, authConfigurationContextKey{}, AuthConfiguration{ProviderConfigured: anyConfigured}), nil
}
