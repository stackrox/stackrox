package idcheck

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
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
