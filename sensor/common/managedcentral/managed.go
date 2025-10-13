package managedcentral

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	managedCentral bool
	managedMutex   sync.RWMutex
)

// IsCentralManaged returns if Central is managed
func IsCentralManaged() bool {
	managedMutex.RLock()
	defer managedMutex.RUnlock()

	return managedCentral
}

// Set sets the value of whether Central is managed
func Set(value bool) {
	managedMutex.Lock()
	defer managedMutex.Unlock()

	managedCentral = value
}
