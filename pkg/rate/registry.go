package rate

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	registry     = make(map[string]*Limiter)
	registryLock sync.RWMutex
)

// RegisterLimiter creates and registers a rate limiter for the given workload name.
// If a limiter already exists for this workload, the existing one is returned.
// This is safe for concurrent access.
func RegisterLimiter(workloadName string, globalRate float64, bucketCapacity int) (*Limiter, error) {
	registryLock.Lock()
	defer registryLock.Unlock()

	if existing, ok := registry[workloadName]; ok {
		return existing, nil
	}

	limiter, err := NewLimiter(workloadName, globalRate, bucketCapacity)
	if err != nil {
		return nil, err
	}
	registry[workloadName] = limiter
	return limiter, nil
}

// GetLimiter returns the rate limiter for the given workload name, or nil if not registered.
func GetLimiter(workloadName string) *Limiter {
	registryLock.RLock()
	defer registryLock.RUnlock()
	return registry[workloadName]
}

// OnSensorDisconnectAll notifies all registered limiters that a sensor has disconnected.
// This should be called when a sensor connection is terminated.
func OnSensorDisconnectAll(sensorID string) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	for _, limiter := range registry {
		limiter.OnSensorDisconnect(sensorID)
	}
}

// ResetForTesting clears the registry. Only use in tests. Required due to use of singleton.
func ResetForTesting() {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry = make(map[string]*Limiter)
}
