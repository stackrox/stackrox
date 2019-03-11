package connection

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Manager is responsible for managing all active connections from sensors.
type Manager interface {
	HandleConnection(clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer) error
	GetConnection(clusterID string) SensorConnection
}
