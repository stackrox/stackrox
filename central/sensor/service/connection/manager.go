package connection

import (
	"time"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
)

type checkInRecorder interface {
	UpdateClusterContactTime(clusterID string, time time.Time) error
}

// Manager is responsible for managing all active connections from sensors.
type Manager interface {
	HandleConnection(clusterID string, pf pipeline.Factory, server central.SensorService_CommunicateServer, recorder checkInRecorder) error
	GetConnection(clusterID string) SensorConnection

	GetActiveConnections() []SensorConnection
}
