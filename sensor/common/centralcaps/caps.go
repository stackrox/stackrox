package centralcaps

import (
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	centralCaps = set.NewSet[centralsensor.CentralCapability]()
	received    bool

	centralCapsMutex sync.RWMutex
)

// Has returns if the connected Central has the given capability.
func Has(cap centralsensor.CentralCapability) bool {
	centralCapsMutex.RLock()
	defer centralCapsMutex.RUnlock()
	return centralCaps.Contains(cap)
}

// Received reports whether Set has been called for this process's Central connection.
func Received() bool {
	centralCapsMutex.RLock()
	defer centralCapsMutex.RUnlock()
	return received
}

// Set records the capabilities advertised in CentralHello.
func Set(caps []centralsensor.CentralCapability) {
	centralCapsMutex.Lock()
	defer centralCapsMutex.Unlock()

	received = true
	centralCaps = set.NewSet(caps...)
}

// Reset clears stored capabilities so Has is false until the next Set.
func Reset() {
	centralCapsMutex.Lock()
	defer centralCapsMutex.Unlock()

	received = false
	centralCaps = set.NewSet[centralsensor.CentralCapability]()
}
