package idcheck

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// A serviceType Authorizer checks that the client has the desired service type.
type serviceType storage.ServiceType

// AuthorizeByIdentity checks whether the TLS identity has the required service context.
func (s serviceType) AuthorizeByIdentity(id authn.Identity) error {
	svc := id.Service()
	if svc == nil {
		return errox.NoCredentials
	}
	if svc.GetType() != storage.ServiceType(s) {
		return errox.NotAuthorized.CausedBy("service source type not allowed")
	}
	return nil
}
