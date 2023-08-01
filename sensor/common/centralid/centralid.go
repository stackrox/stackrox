package centralid

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	centralID string
	mutex     sync.RWMutex
)

// Get returns the ID of the connected Central.
func Get() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return centralID
}

// Set sets the Central ID.
func Set(value string) {
	mutex.Lock()
	defer mutex.Unlock()
	centralID = value
}
