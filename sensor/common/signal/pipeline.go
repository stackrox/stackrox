package signal

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
)

// Pipeline defines the way to process a process signal
type Pipeline interface {
	Process(signal *storage.ProcessSignal)
	Shutdown()
	Notify(e common.SensorComponentEvent)
}
