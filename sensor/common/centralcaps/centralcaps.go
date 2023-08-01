package centralcaps

import (
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	centralCaps = set.NewSet[centralsensor.CentralCapability]()

	centralCapsMutex sync.RWMutex
)

// Has returns if the Central has the given capability.
func Has(cap centralsensor.CentralCapability) bool {
	centralCapsMutex.RLock()
	defer centralCapsMutex.RUnlock()
	return centralCaps.Contains(cap)
}

// Set sets the capabilities of the connected Central.
func Set(caps []centralsensor.CentralCapability) {
	centralCapsMutex.Lock()
	defer centralCapsMutex.Unlock()

	centralCaps = set.NewSet(caps...)
}
