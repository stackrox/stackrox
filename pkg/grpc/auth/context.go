package auth

import (
	"context"
	"errors"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/mtls"
)

var (
	// ErrNoContext is returned when we process a context, but can't find any Identity info.
	ErrNoContext = errors.New("no identity context found")
)

type tlsContextKey struct{}
type userContextKey struct{}

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

// A UserIdentity holds an identity extracted from a user authentication token.
type UserIdentity struct {
	authproviders.User
	AuthProvider authproviders.Authenticator
	Expiration   time.Time
}

// NewUserContext adds the given Identity to the Context.
func NewUserContext(ctx context.Context, id UserIdentity) context.Context {
	return context.WithValue(ctx, userContextKey{}, id)
}

// FromUserContext retrieves identity information from the given context.
// The context must have been passed through the interceptors provided by this package.
func FromUserContext(ctx context.Context) (UserIdentity, error) {
	val, ok := ctx.Value(userContextKey{}).(UserIdentity)
	if !ok {
		return UserIdentity{}, ErrNoContext
	}
	return val, nil
}
