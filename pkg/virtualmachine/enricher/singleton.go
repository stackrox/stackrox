package enricher

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vmEnricherOnce     sync.Once
	vmEnricherInstance VirtualMachineEnricher
	vmLog              = logging.LoggerForModule()
)

func Singleton() VirtualMachineEnricher {
	vmEnricherOnce.Do(func() {
		vmScanner, err := scannerv4.NewVirtualMachineScanner()
		if err != nil {
			vmLog.Errorf("Failed to create Scanner V4 for VM enricher: %v", err)
			// Return enricher with nil client - it will handle errors gracefully
			vmEnricherInstance = New(nil)
			return
		}
		vmEnricherInstance = New(vmScanner)
	})
	return vmEnricherInstance
}
