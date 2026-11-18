package idcheck

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// IdentityBasedAuthorizer is an authorizer based on identity.
type IdentityBasedAuthorizer interface {
	AuthorizeByIdentity(identity authn.Identity) error
}

type identityBasedAuthorizerWrapper struct {
	idAuthorizer IdentityBasedAuthorizer
}

// Wrap wraps an IdentityBasedAuthorizer to conform to the authz.Authorizer interface.
func Wrap(idAuthorizer IdentityBasedAuthorizer) authz.Authorizer {
	return identityBasedAuthorizerWrapper{idAuthorizer: idAuthorizer}
}

// EnrichContext returns the input context unchanged.
func (w identityBasedAuthorizerWrapper) EnrichContext(ctx context.Context, _ string) context.Context {
	return ctx
}

// Authorized implements the Authorizer interface.
func (w identityBasedAuthorizerWrapper) Authorized(ctx context.Context, _ string) error {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}
	return w.idAuthorizer.AuthorizeByIdentity(id)
}
