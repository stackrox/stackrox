package idcheck

import (
	"context"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// A serviceType Authorizer checks that the client has the desired service type.
type serviceType struct {
	Type v1.ServiceType
}

// Authorized checks whether the TLS identity has the required service context.
func (s serviceType) Authorized(ctx context.Context, _ string) error {
	identity, err := authn.FromTLSContext(ctx)
	if err != nil {
		return authz.ErrNoCredentials
	}
	if identity.Subject.ServiceType != s.Type {
		return authz.ErrNotAuthorized("service source type not allowed")
	}
	return nil
}
