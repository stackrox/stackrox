package idcheck

import (
	"github.com/stackrox/rox/pkg/grpc/authn"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// A serviceType Authorizer checks that the client has the desired service type.
type serviceType v1.ServiceType

// Authorized checks whether the TLS identity has the required service context.
func (s serviceType) AuthorizeByIdentity(id authn.Identity) error {
	svc := id.Service()
	if svc == nil {
		return authz.ErrNoCredentials
	}
	if svc.GetType() != v1.ServiceType(s) {
		return authz.ErrNotAuthorized("service source type not allowed")
	}
	return nil
}
