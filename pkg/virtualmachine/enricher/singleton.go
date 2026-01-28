package enricher

import (
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vmEnricherOnce     sync.Once
	vmEnricherInstance VirtualMachineEnricher
)

func Singleton() VirtualMachineEnricher {
	vmEnricherOnce.Do(func() {
		vmEnricherInstance = newWithCreator(scannerv4.VirtualMachineScannerCreator)
	})
	return vmEnricherInstance
}
