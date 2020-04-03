package authn

import (
	"context"
	"testing"
)

type identityContextKey struct{}

// IdentityFromContext retrieves the identity from the context, if any.
func IdentityFromContext(ctx context.Context) Identity {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	return id
}

// ContextWithIdentity adds the given identity to the context. It is to be used only for testing --
// to help remind users of this, it takes in a testing.T.
func ContextWithIdentity(ctx context.Context, identity Identity, _ *testing.T) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}
