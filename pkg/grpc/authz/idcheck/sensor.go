package idcheck

import (
	"context"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
)

// A ServiceType Authorizer checks that the client has the desired service type.
type ServiceType struct {
	Type v1.ServiceType
}

// Authorized checks whether the TLS identity has the required service context.
func (s ServiceType) Authorized(ctx context.Context) error {
	identity, err := authn.FromTLSContext(ctx)
	if err != nil {
		return authz.ErrNoCredentials{}
	}
	if identity.Name.ServiceType != v1.ServiceType_SENSOR_SERVICE {
		return authz.ErrNotAuthorized{Explanation: "only sensors are allowed"}
	}
	return nil
}

// SensorsOnly returns a ServiceType authorizer that checks for the Sensor type.
func SensorsOnly() ServiceType {
	return ServiceType{Type: v1.ServiceType_SENSOR_SERVICE}
}
