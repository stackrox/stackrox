package manager

import (
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/timestamp"
	"github.com/stackrox/stackrox/sensor/common"
)

var (
	log = logging.LoggerForModule()
)

// Manager processes network connections coming in from collector, enriches them and sends them to Central
type Manager interface {
	UnregisterCollector(hostname string, sequenceID int64)
	RegisterCollector(hostname string) (HostNetworkInfo, int64)

	PublicIPsValueStream() concurrency.ReadOnlyValueStream
	ExternalSrcsValueStream() concurrency.ReadOnlyValueStream

	common.SensorComponent
}

// HostNetworkInfo processes network connections from a single host aka collector.
type HostNetworkInfo interface {
	Process(networkInfo *sensor.NetworkConnectionInfo, nowTimestamp timestamp.MicroTS, sequenceID int64) error
}
