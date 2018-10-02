package manager

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Manager processes network connections coming in from collector, enriches them and sends them to Central
type Manager interface {
	Start()
	Stop()

	RegisterCollector(hostname string) HostNetworkInfo
}

// HostNetworkInfo processes network connections from a single host aka collector.
type HostNetworkInfo interface {
	Process(networkInfo *sensor.NetworkConnectionInfo, currTimestamp time.Time, isFirst bool)
}
