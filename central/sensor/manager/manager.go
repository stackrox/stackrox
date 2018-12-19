package manager

import "github.com/stackrox/rox/generated/internalapi/central"

// SensorConnection is the interface for active connections from a sensor instance.
type SensorConnection interface {
	Communicate(server central.SensorService_CommunicateServer) error
}

// SensorManager keeps track of connections to sensor instances.
type SensorManager interface {
	CreateConnection(clusterID string) (SensorConnection, error)
	RemoveConnection(clusterID string, connection SensorConnection) error
}
