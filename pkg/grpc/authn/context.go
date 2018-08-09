package authn

import (
	"context"
	"errors"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokenbased"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	// ErrNoContext is returned when we process a context, but can't find any Identity info.
	ErrNoContext = errors.New("no identity context found")
)

type tlsContextKey struct{}
type tokenBasedIdentityContextKey struct{}
type authConfigurationContextKey struct{}

// A TLSIdentity holds an identity extracted from service-to-service TLS credentials.
type TLSIdentity struct {
	mtls.Identity
	Expiration time.Time
}

// NewTLSContext adds the given Identity to the Context.
func NewTLSContext(ctx context.Context, id TLSIdentity) context.Context {
	return context.WithValue(ctx, tlsContextKey{}, id)
}

// FromTLSContext retrieves identity information from the given context.
// The context must have been passed through the interceptors provided by this package.
func FromTLSContext(ctx context.Context) (TLSIdentity, error) {
	val, ok := ctx.Value(tlsContextKey{}).(TLSIdentity)
	if !ok {
		return TLSIdentity{}, ErrNoContext
	}
	return val, nil
}

// NewTokenBasedIdentityContext adds the given Identity to the Context.
func NewTokenBasedIdentityContext(ctx context.Context, id tokenbased.Identity) context.Context {
	return context.WithValue(ctx, tokenBasedIdentityContextKey{}, id)
}

// FromTokenBasedIdentityContext retrieves identity information from the given context.
// The context must have been passed through the interceptors provided by this package.
func FromTokenBasedIdentityContext(ctx context.Context) (tokenbased.Identity, error) {
	val, ok := ctx.Value(tokenBasedIdentityContextKey{}).(tokenbased.Identity)
	if !ok {
		return nil, ErrNoContext
	}
	return val, nil
}

// AuthConfiguration provides information about how auth is configured (or not).
type AuthConfiguration struct {
	// ProviderConfigured indicates at least one provider is configured.
	ProviderConfigured bool
}

// NewAuthConfigurationContext adds the given AuthConfiguration to the Context.
func NewAuthConfigurationContext(ctx context.Context, conf AuthConfiguration) context.Context {
	return context.WithValue(ctx, authConfigurationContextKey{}, conf)
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
