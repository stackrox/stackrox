package authn

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errorhelpers"
)

type identityContextKey struct{}
type identityErrorContextKey struct{}

// IdentityFromContextOrError retrieves the identity from the context or returns error otherwise.
func IdentityFromContextOrError(ctx context.Context) (Identity, error) {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	err, _ := ctx.Value(identityErrorContextKey{}).(error)
	switch {
	case err != nil:
		return nil, errorhelpers.NewErrNoCredentials(err.Error())
	case id == nil && err == nil:
		return nil, errorhelpers.ErrNoCredentials
	default:
		return id, nil
	}
}

// IdentityFromContextOrNil retrieves the identity from the context, if any.
func IdentityFromContextOrNil(ctx context.Context) Identity {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	return id
}

// ContextWithIdentity adds the given identity to the context. It is to be used only for testing --
// to help remind users of this, it takes in a testing.T.
func ContextWithIdentity(ctx context.Context, identity Identity, _ *testing.T) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}
