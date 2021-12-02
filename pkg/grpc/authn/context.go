package authn

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errorhelpers"
)

type identityContextKey struct{}
type identityErrorContextKey struct{}

// IdentityFromContext retrieves the identity from the context, if any.
func IdentityFromContext(ctx context.Context) (Identity, error) {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	err, _ := ctx.Value(identityErrorContextKey{}).(error)
	if id == nil && err == nil {
		return nil, errorhelpers.ErrNoCredentials
	}
	return id, err
}

// ContextWithIdentity adds the given identity to the context. It is to be used only for testing --
// to help remind users of this, it takes in a testing.T.
func ContextWithIdentity(ctx context.Context, identity Identity, _ *testing.T) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}
