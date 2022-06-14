package signal

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Pipeline defines the way to process a process signal
type Pipeline interface {
	Process(signal *storage.ProcessSignal)
}
