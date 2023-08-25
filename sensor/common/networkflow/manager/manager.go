package manager

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
)

var (
	log = logging.LoggerForModule()
)

// Manager processes network connections coming in from collector, enriches them and sends them to Central
type Manager interface {
	UnregisterCollector(hostname string, sequenceID int64)
	RegisterCollector(hostname string) (HostNetworkInfo, int64)

	PublicIPsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPAddressList]
	ExternalSrcsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPNetworkList]

	common.SensorComponent
}

// HostNetworkInfo processes network connections from a single host aka collector.
type HostNetworkInfo interface {
	Process(networkInfo *sensor.NetworkConnectionInfo, nowTimestamp timestamp.MicroTS, sequenceID int64) error
}
