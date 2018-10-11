package authn

import "context"

type identityContextKey struct{}

// IdentityFromContext retrieves the identity from the context, if any.
func IdentityFromContext(ctx context.Context) Identity {
	id, _ := ctx.Value(identityContextKey{}).(Identity)
	return id
}
