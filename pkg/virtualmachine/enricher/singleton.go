package enricher

import (
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vmEnricherOnce     sync.Once
	vmEnricherInstance VirtualMachineEnricher
)

// Singleton returns the shared VM enricher instance used by both Central
// integration lifecycle wiring and the VM-index pipeline.
// Sharing one instance matters because explicit VM-scanner integrations are
// stateful and must be observed consistently everywhere VM enrichment happens.
func Singleton(resolveScanner func() types.VirtualMachineScanner) VirtualMachineEnricher {
	vmEnricherOnce.Do(func() {
		vmEnricherInstance = New(resolveScanner)
	})
	return vmEnricherInstance
}
