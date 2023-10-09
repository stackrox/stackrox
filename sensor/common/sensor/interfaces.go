package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// ServiceCommunicateClient is used to generate the mocks for testing the gRPC client
//
//go:generate mockgen-wrapper
type ServiceCommunicateClient interface {
	central.SensorService_CommunicateClient
}
