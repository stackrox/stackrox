package idcheck

import (
	"context"

	"github.com/stackrox/rox/pkg/errox"
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

// Authorized implements the Authorizer interface.
func (w identityBasedAuthorizerWrapper) Authorized(ctx context.Context, _ string) error {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}
	return w.idAuthorizer.AuthorizeByIdentity(id)
}

type anyServiceAuthorizer struct{}

// AnyService returns an authorizer that allows any service identity.
func AnyService() authz.Authorizer {
	return anyServiceAuthorizer{}
}

// Authorized implements the Authorizer interface.
func (a anyServiceAuthorizer) Authorized(ctx context.Context, _ string) error {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}
	svc := id.Service()
	if svc == nil {
		return errox.NoCredentials
	}
	return nil
}
