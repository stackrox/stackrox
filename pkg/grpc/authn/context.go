package authn

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
)

type identityContextKey struct{}
type identityErrorContextKey struct{}

// IdentityFromContext retrieves the identity from the context or returns error otherwise.
func IdentityFromContext(ctx context.Context) (Identity, error) {
	err, _ := ctx.Value(identityErrorContextKey{}).(error)
	if err != nil {
		return nil, err
	}
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	if id != nil {
		return id, nil
	}
	return nil, errox.NoCredentials
}

// IdentityFromContextOrNil retrieves the identity from the context, if any.
func IdentityFromContextOrNil(ctx context.Context) Identity {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	return id
}

// ContextWithIdentity adds the given identity to the context. It is to be used only for testing --
// to help remind users of this, it takes in a testing.T.
func ContextWithIdentity(ctx context.Context, identity Identity, _ testing.TB) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}
